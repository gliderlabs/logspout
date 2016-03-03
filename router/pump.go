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

	"github.com/fsouza/go-dockerclient"
)

func init() {
	pump := &LogsPump{
		pumps:  make(map[string]*containerPump),
		routes: make(map[chan *update]struct{}),
	}
	LogRouters.Register(pump, "pump")
	Jobs.Register(pump, "pump")
}

func getopt(name, dfault string) string {
	value := os.Getenv(name)
	if value == "" {
		value = dfault
	}
	return value
}

func debug(v ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		log.Println(v...)
	}
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

func ignoreContainer(container *docker.Container) bool {
	for _, kv := range container.Config.Env {
		kvp := strings.SplitN(kv, "=", 2)
		if len(kvp) == 2 && kvp[0] == "LOGSPOUT" && strings.ToLower(kvp[1]) == "ignore" {
			return true
		}
	}
	return false
}

type update struct {
	*docker.APIEvents
	pump *containerPump
}

type LogsPump struct {
	mu     sync.Mutex
	pumps  map[string]*containerPump
	routes map[chan *update]struct{}
	client *docker.Client
}

func (p *LogsPump) Name() string {
	return "pump"
}

func (p *LogsPump) Setup() error {
	client, err := docker.NewClient(
		getopt("DOCKER_HOST", "unix:///var/run/docker.sock"))
	if err != nil {
		return err
	}
	p.client = client
	return nil
}

func (p *LogsPump) Run() error {
	containers, err := p.client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return err
	}
	for _, listing := range containers {
		p.pumpLogs(&docker.APIEvents{
			ID:     normalID(listing.ID),
			Status: "start",
		}, false)
	}
	events := make(chan *docker.APIEvents)
	err = p.client.AddEventListener(events)
	if err != nil {
		return err
	}
	for event := range events {
		debug("pump: event:", normalID(event.ID), event.Status)
		switch event.Status {
		case "start", "restart":
			go p.pumpLogs(event, true)
		case "die":
			go p.update(event)
		}
	}
	return errors.New("docker event stream closed")
}

func (p *LogsPump) pumpLogs(event *docker.APIEvents, backlog bool) {
	id := normalID(event.ID)
	container, err := p.client.InspectContainer(id)
	assert(err, "pump")
	if container.Config.Tty {
		debug("pump:", id, "ignored: tty enabled")
		return
	}
	if ignoreContainer(container) {
		debug("pump:", id, "ignored: environ ignore")
		return
	}
	var tail string
	if backlog {
		tail = "all"
	} else {
		tail = "0"
	}
	outrd, outwr := io.Pipe()
	errrd, errwr := io.Pipe()
	p.mu.Lock()
	p.pumps[id] = newContainerPump(container, outrd, errrd)
	p.mu.Unlock()
	p.update(event)
	debug("pump:", id, "started")
	go func() {
		err := p.client.Logs(docker.LogsOptions{
			Container:    id,
			OutputStream: outwr,
			ErrorStream:  errwr,
			Stdout:       true,
			Stderr:       true,
			Follow:       true,
			Tail:         tail,
		})
		if err != nil {
			debug("pump:", id, "stopped:", err)
		}
		outwr.Close()
		errwr.Close()
		p.mu.Lock()
		delete(p.pumps, id)
		p.mu.Unlock()
	}()
}

func (p *LogsPump) update(event *docker.APIEvents) {
	p.mu.Lock()
	defer p.mu.Unlock()
	pump, pumping := p.pumps[normalID(event.ID)]
	if pumping {
		for r, _ := range p.routes {
			select {
			case r <- &update{event, pump}:
			case <-time.After(time.Second * 1):
				debug("pump: route timeout, dropping")
				defer delete(p.routes, r)
			}
		}
	}
}

func (p *LogsPump) RoutingFrom(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, monitoring := p.pumps[normalID(id)]
	return monitoring
}

func (p *LogsPump) Route(route *Route, logstream chan *Message) {
	p.mu.Lock()
	for _, pump := range p.pumps {
		if route.MatchContainer(
			normalID(pump.container.ID),
			normalName(pump.container.Name)) {

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
	}()
	for {
		select {
		case event := <-updates:
			switch event.Status {
			case "start", "restart":
				if route.MatchContainer(
					normalID(event.pump.container.ID),
					normalName(event.pump.container.Name)) {

					event.pump.add(logstream, route)
					defer event.pump.remove(logstream)
				}
			case "die":
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
					debug("pump:", normalID(container.ID), source+":", err)
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
		select {
		case logstream <- msg:
		case <-time.After(time.Second * 1):
			debug("pump: send timeout, closing")
			// normal call to remove() triggered by
			// route.Closer() may not be able to grab
			// lock under heavy load, so we delete here
			defer delete(cp.logstreams, logstream)
		}
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
