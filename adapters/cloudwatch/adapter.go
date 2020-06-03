package cloudwatch

import (
	"bytes"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/gliderlabs/logspout/router"
)

func init() {
	router.AdapterFactories.Register(NewAdapter, "cloudwatch")
}

const defaultMaxRetries = 5

// Adapter is an adapter that streams JSON to AWS CloudwatchLogs.
// It mostly just checkes ENV vars and other container info to determine
// the LogGroup and LogStream for each message, then sends each message
// on to a CloudwatchBatcher, which batches messages for upload to AWS.
type Adapter struct {
	Route       *router.Route
	OsHost      string
	Ec2Region   string
	Ec2Instance string
	maxRetries  int

	client      *docker.Client
	batcher     *Batcher          // batches up messages by log group and stream
	groupnames  map[string]string // maps container names to log groups
	streamnames map[string]string // maps container names to log streams
}

// NewAdapter creates a CloudwatchAdapter for the current region.
func NewAdapter(route *router.Route) (router.LogAdapter, error) {
	maxRetries := defaultMaxRetries
	if envVal := os.Getenv(`MAX_RETRIES`); envVal != "" {
		i, err := strconv.Atoi(envVal)
		if err != nil {
			return nil, err
		}
		maxRetries = i
	}
	dockerHost := `unix:///var/run/docker.sock`
	if envVal := os.Getenv(`DOCKER_HOST`); envVal != "" {
		dockerHost = envVal
	}
	client, err := docker.NewClient(dockerHost)
	if err != nil {
		return nil, err
	}
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	ec2info, err := NewEC2Info(route) // get info from EC2
	if err != nil {
		return nil, err
	}
	adapter := Adapter{
		Route:       route,
		OsHost:      hostname,
		Ec2Instance: ec2info.InstanceID,
		Ec2Region:   ec2info.Region,
		maxRetries:  maxRetries,
		client:      client,
		groupnames:  map[string]string{},
		streamnames: map[string]string{},
	}
	adapter.batcher = NewCloudwatchBatcher(&adapter)
	return &adapter, nil
}

// Stream implements the router.LogAdapter interface.
func (a *Adapter) Stream(logstream chan *router.Message) {
	for m := range logstream {
		// determine the log group name and log stream name
		var groupName, streamName string
		// first, check the in-memory cache so this work is done per-container
		if cachedGroup, isCached := a.groupnames[m.Container.ID]; isCached {
			groupName = cachedGroup
		}
		if cachedStream, isCached := a.streamnames[m.Container.ID]; isCached {
			streamName = cachedStream
		}
		if (streamName == "") || (groupName == "") {
			// make a render context with the required info
			containerData, err := a.client.InspectContainer(m.Container.ID)
			if err != nil {
				log.Println("cloudwatch: error inspecting container:", err)
				continue
			}
			context := RenderContext{
				Env:        parseEnv(m.Container.Config.Env),
				Labels:     containerData.Config.Labels,
				Name:       strings.TrimPrefix(m.Container.Name, `/`),
				ID:         m.Container.ID,
				Host:       m.Container.Config.Hostname,
				LoggerHost: a.OsHost,
				InstanceID: a.Ec2Instance,
				Region:     a.Ec2Region,
			}
			groupName = a.renderEnvValue(`LOGSPOUT_GROUP`, &context, a.OsHost)
			streamName = a.renderEnvValue(`LOGSPOUT_STREAM`, &context, context.Name)
			a.groupnames[m.Container.ID] = groupName   // cache the group name
			a.streamnames[m.Container.ID] = streamName // and the stream name
		}
		a.batcher.Input <- Message{
			Message:   m.Data,
			Group:     groupName,
			Stream:    streamName,
			Time:      time.Now(),
			Container: m.Container.ID,
		}
	}
}

// Searches the OS environment, then the route options, then the render context
// Env for a given key, then uses the value (or the provided default value)
// as template text, which is then rendered in the given context.
// The rendered result is returned - or the default value on any errors.
func (a *Adapter) renderEnvValue(
	envKey string, context *RenderContext, defaultVal string) string {
	finalVal := defaultVal
	if logspoutEnvVal := os.Getenv(envKey); logspoutEnvVal != "" {
		finalVal = logspoutEnvVal // use $envKey, if set
	}
	if routeOptionsVal, exists := a.Route.Options[envKey]; exists {
		finalVal = routeOptionsVal
	}
	if containerEnvVal, exists := context.Env[envKey]; exists {
		finalVal = containerEnvVal // or, $envKey from container!
	}
	template, err := template.New("template").Parse(finalVal)
	if err != nil {
		log.Println("cloudwatch: error parsing template", finalVal, ":", err)
		return defaultVal
	}
	// render the templates in the generated context
	var renderedValue bytes.Buffer
	err = template.Execute(&renderedValue, context)
	if err != nil {
		log.Printf("cloudwatch: error rendering template %s : %s\n",
			finalVal, err)
		return defaultVal
	}
	return renderedValue.String()
}

func parseEnv(envLines []string) map[string]string {
	env := map[string]string{}
	for _, line := range envLines {
		fields := strings.Split(line, `=`)
		if len(fields) > 1 {
			env[fields[0]] = strings.Join(fields[1:], `=`)
		}
	}
	return env
}
