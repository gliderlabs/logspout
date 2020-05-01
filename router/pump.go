package router

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/gliderlabs/logspout/cfg"
)

const (
	defaultPumpName            = "pump"
	pumpEventStatusStartName   = "start"
	pumpEventStatusRestartName = "restart"
	pumpEventStatusRenameName  = "rename"
	pumpEventStatusDieName     = "die"
	trueString                 = "true"
)

var (
	allowTTY bool
)

func init() {
	pump := &LogsPump{
		pumps:  make(map[string]*containerPump),
		routes: make(map[chan *update]struct{}),
	}
	setAllowTTY()
	LogRouters.Register(pump, defaultPumpName)
	Jobs.Register(pump, defaultPumpName)
}

func debug(v ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		log.Println(v...)
	}
}

func backlog() bool {
	return os.Getenv("BACKLOG") == "false"
}

func setAllowTTY() {
	if t := cfg.GetEnvDefault("ALLOW_TTY", ""); t == trueString {
		allowTTY = true
	}
	debug("setting allowTTY to:", allowTTY)
}

func assert(err error, context string) {
	if err != nil {
		log.Fatal(context+": ", err)
	}
}

func normalName(name string) string {
	return name[1:]
}

func normalID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func logDriverSupported(container *docker.Container) bool {
	switch container.HostConfig.LogConfig.Type {
	case "json-file", "journald", "db":
		return true
	default:
		return false
	}
}

func ignoreContainer(container *docker.Container) bool {
	for _, kv := range container.Config.Env {
		kvp := strings.SplitN(kv, "=", 2)
		if len(kvp) == 2 && kvp[0] == "LOGSPOUT" && strings.EqualFold(kvp[1], "ignore") {
			return true
		}
	}

	excludeLabel := cfg.GetEnvDefault("EXCLUDE_LABELS", "")

	if excludeLabel == "" {
		excludeLabel = cfg.GetEnvDefault("EXCLUDE_LABEL", "")
	}
	excludeValue := "true"
	// support EXCLUDE_LABEL having multiple custom label values
	excludeLabelArr := strings.Split(excludeLabel, ";")

	for _, label := range excludeLabelArr {
		labelParts := strings.Split(label, ":")

		if len(labelParts) == 2 {
			excludeLabel = labelParts[0]
			excludeValue = labelParts[1]
		}

		if value, ok := container.Config.Labels[excludeLabel]; ok {
			if len(excludeLabel) > 0 && strings.EqualFold(value, strings.ToLower(excludeValue)) {
				return true
			}
		}
	}
	return false
}

func ignoreContainerTTY(container *docker.Container) bool {
	if container.Config.Tty && !allowTTY {
		return true
	}
	return false
}

func getInactivityTimeoutFromEnv() time.Duration {
	inactivityTimeout, err := time.ParseDuration(cfg.GetEnvDefault("INACTIVITY_TIMEOUT", "0"))
	assert(err, "Couldn't parse env var INACTIVITY_TIMEOUT. See https://golang.org/pkg/time/#ParseDuration for valid format.")
	return inactivityTimeout
}

type update struct {
	*docker.APIEvents
	pump *containerPump
}

// LogsPump is responsible for "pumping" logs to their configured destinations
type LogsPump struct {
	mu     sync.Mutex
	pumps  map[string]*containerPump
	routes map[chan *update]struct{}
	client *docker.Client
}

// Name returns the name of the pump
func (p *LogsPump) Name() string {
	return defaultPumpName
}

// Setup configures the pump
func (p *LogsPump) Setup() error {
	var err error
	p.client, err = docker.NewClientFromEnv()
	return err
}

func (p *LogsPump) rename(event *docker.APIEvents) {
	p.mu.Lock()
	defer p.mu.Unlock()
	container, err := p.client.InspectContainer(event.ID)
	assert(err, defaultPumpName)
	pump, ok := p.pumps[normalID(event.ID)]
	if !ok {
		debug("pump.rename(): ignore: pump not found, state:", container.State.StateString())
		return
	}
	pump.container.Name = container.Name
}

// Run executes the pump
func (p *LogsPump) Run() error {
	inactivityTimeout := getInactivityTimeoutFromEnv()
	debug("pump.Run(): using inactivity timeout: ", inactivityTimeout)

	containers, err := p.client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return err
	}
	for idx := range containers {
		p.pumpLogs(&docker.APIEvents{
			ID:     normalID(containers[idx].ID),
			Status: pumpEventStatusStartName,
		}, false, inactivityTimeout)
	}
	events := make(chan *docker.APIEvents)
	err = p.client.AddEventListener(events)
	if err != nil {
		return err
	}
	for event := range events {
		debug("pump.Run() event:", normalID(event.ID), event.Status)
		switch event.Status {
		case pumpEventStatusStartName, pumpEventStatusRestartName:
			go p.pumpLogs(event, backlog(), inactivityTimeout)
		case pumpEventStatusRenameName:
			go p.rename(event)
		case pumpEventStatusDieName:
			go p.update(event)
		}
	}
	return errors.New("docker event stream closed")
}

func (p *LogsPump) pumpLogs(event *docker.APIEvents, backlog bool, inactivityTimeout time.Duration) { //nolint:gocyclo
	id := normalID(event.ID)
	container, err := p.client.InspectContainer(id)
	assert(err, defaultPumpName)
	if ignoreContainerTTY(container) {
		debug("pump.pumpLogs():", id, "ignored: tty enabled")
		return
	}
	if ignoreContainer(container) {
		debug("pump.pumpLogs():", id, "ignored: environ ignore")
		return
	}
	if !logDriverSupported(container) {
		debug("pump.pumpLogs():", id, "ignored: log driver not supported")
		return
	}

	var tail = cfg.GetEnvDefault("TAIL", "all")
	var sinceTime time.Time
	if backlog {
		sinceTime = time.Unix(0, 0)
	} else {
		sinceTime = time.Now()
	}

	p.mu.Lock()
	if _, exists := p.pumps[id]; exists {
		p.mu.Unlock()
		debug("pump.pumpLogs():", id, "pump exists")
		return
	}

	// RawTerminal with container Tty=false injects binary headers into
	// the log stream that show up as garbage unicode characters
	rawTerminal := false
	if allowTTY && container.Config.Tty {
		rawTerminal = true
	}
	outrd, outwr := io.Pipe()
	errrd, errwr := io.Pipe()
	p.pumps[id] = newContainerPump(container, outrd, errrd)
	p.mu.Unlock()
	p.update(event)
	go func() {
		for {
			debug("pump.pumpLogs():", id, "started, tail:", tail)
			err := p.client.Logs(docker.LogsOptions{
				Container:         id,
				OutputStream:      outwr,
				ErrorStream:       errwr,
				Stdout:            true,
				Stderr:            true,
				Follow:            true,
				Tail:              tail,
				Since:             sinceTime.Unix(),
				InactivityTimeout: inactivityTimeout,
				RawTerminal:       rawTerminal,
			})
			if err != nil {
				debug("pump.pumpLogs():", id, "stopped with error:", err)
			} else {
				debug("pump.pumpLogs():", id, "stopped")
			}

			sinceTime = time.Now()
			if err == docker.ErrInactivityTimeout {
				sinceTime = sinceTime.Add(-inactivityTimeout)
			}

			container, err := p.client.InspectContainer(id)
			if err != nil {
				_, four04 := err.(*docker.NoSuchContainer)
				if !four04 {
					assert(err, defaultPumpName)
				}
			} else if container.State.Running {
				continue
			}

			debug("pump.pumpLogs():", id, "dead")
			outwr.Close()
			errwr.Close()
			p.mu.Lock()
			delete(p.pumps, id)
			p.mu.Unlock()
			return
		}
	}()
}

func (p *LogsPump) update(event *docker.APIEvents) {
	p.mu.Lock()
	defer p.mu.Unlock()
	pump, pumping := p.pumps[normalID(event.ID)]
	if pumping {
		for r := range p.routes {
			select {
			case r <- &update{event, pump}:
			case <-time.After(time.Second * 1):
				debug("pump.update(): route timeout, dropping")
				defer delete(p.routes, r)
			}
		}
	}
}

// RoutingFrom returns whether a container id is routing from this pump
func (p *LogsPump) RoutingFrom(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, monitoring := p.pumps[normalID(id)]
	return monitoring
}

// Route takes a logstream and routes it according to the supplied Route
func (p *LogsPump) Route(route *Route, logstream chan *Message) {
	p.mu.Lock()
	for _, pump := range p.pumps {
		if route.MatchContainer(
			normalID(pump.container.ID),
			normalName(pump.container.Name),
			pump.container.Config.Labels) {

			pump.add(logstream, route)
			defer pump.remove(logstream)
		}
	}
	updates := make(chan *update)
	p.routes[updates] = struct{}{}
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		delete(p.routes, updates)
		p.mu.Unlock()
		route.closed = true
	}()
	for {
		select {
		case event := <-updates:
			switch event.Status {
			case pumpEventStatusStartName, pumpEventStatusRestartName:
				if route.MatchContainer(
					normalID(event.pump.container.ID),
					normalName(event.pump.container.Name),
					event.pump.container.Config.Labels) {

					event.pump.add(logstream, route)
					defer event.pump.remove(logstream)
				}
			case pumpEventStatusDieName:
				if strings.HasPrefix(route.FilterID, event.ID) {
					// If the route is just about a single container,
					// we can stop routing when it dies.
					return
				}
			}
		case <-route.Closer():
			return
		}
	}
}

type containerPump struct {
	sync.Mutex
	container  *docker.Container
	logstreams map[chan *Message]*Route
}

func newContainerPump(container *docker.Container, stdout, stderr io.Reader) *containerPump {
	cp := &containerPump{
		container:  container,
		logstreams: make(map[chan *Message]*Route),
	}
	pump := func(source string, input io.Reader) {
		buf := bufio.NewReader(input)
		for {
			line, err := buf.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					debug("pump.newContainerPump():", normalID(container.ID), source+":", err)
				}
				return
			}
			cp.send(&Message{
				Data:      strings.TrimSuffix(line, "\n"),
				Container: container,
				Time:      time.Now(),
				Source:    source,
			})
		}
	}
	go pump("stdout", stdout)
	go pump("stderr", stderr)
	return cp
}

func (cp *containerPump) send(msg *Message) {
	cp.Lock()
	defer cp.Unlock()
	for logstream, route := range cp.logstreams {
		if !route.MatchMessage(msg) {
			continue
		}
		logstream <- msg
	}
}

func (cp *containerPump) add(logstream chan *Message, route *Route) {
	cp.Lock()
	defer cp.Unlock()
	cp.logstreams[logstream] = route
}

func (cp *containerPump) remove(logstream chan *Message) {
	cp.Lock()
	defer cp.Unlock()
	delete(cp.logstreams, logstream)
}
