package http

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gliderlabs/logspout/router"
)

func init() {
	router.AdapterFactories.Register(NewHTTPAdapter, "http")
	router.AdapterFactories.Register(NewHTTPAdapter, "https")
}

func debug(v ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		log.Println(v...)
	}
}

func die(v ...interface{}) {
	panic(fmt.Sprintln(v...))
}

func getopt(name, dfault string) string {
	value := os.Getenv(name)
	if value == "" {
		value = dfault
	}
	return value
}

func getStringParameter(
	options map[string]string, parameterName string, dfault string) string {

	if value, ok := options[parameterName]; ok {
		return value
	} else {
		return dfault
	}
}

func getIntParameter(
	options map[string]string, parameterName string, dfault int) int {

	if value, ok := options[parameterName]; ok {
		valueInt, err := strconv.Atoi(value)
		if err != nil {
			debug("http: invalid value for parameter:", parameterName, value)
			return dfault
		} else {
			return valueInt
		}
	} else {
		return dfault
	}
}

func getDurationParameter(
	options map[string]string, parameterName string,
	dfault time.Duration) time.Duration {

	if value, ok := options[parameterName]; ok {
		valueDuration, err := time.ParseDuration(value)
		if err != nil {
			debug("http: invalid value for parameter:", parameterName, value)
			return dfault
		} else {
			return valueDuration
		}
	} else {
		return dfault
	}
}

func dial(netw, addr string) (net.Conn, error) {
	dial, err := net.Dial(netw, addr)
	if err != nil {
		debug("http: new dial", dial, err, netw, addr)
	} else {
		debug("http: new dial", dial, netw, addr)
	}
	return dial, err
}

// HTTPAdapter is an adapter that POSTs logs to an HTTP endpoint
type HTTPAdapter struct {
	route             *router.Route
	url               string
	client            *http.Client
	buffer            []*router.Message
	timer             *time.Timer
	capacity          int
	timeout           time.Duration
	totalMessageCount int
	bufferMutex       sync.Mutex
}

// NewHTTPAdapter creates an HTTPAdapter
func NewHTTPAdapter(route *router.Route) (router.LogAdapter, error) {

	// Figure out the URI and create the HTTP client
	defaultPath := ""
	path := getStringParameter(route.Options, "http.path", defaultPath)
	url := fmt.Sprintf("%s://%s%s", route.Adapter, route.Address, path)
	debug("http: url:", url)
	transport := &http.Transport{}
	transport.Dial = dial
	client := &http.Client{Transport: transport}

	// Determine the buffer capacity
	defaultCapacity := 100
	capacity := getIntParameter(
		route.Options, "http.buffer.capacity", defaultCapacity)
	if capacity < 1 || capacity > 10000 {
		debug("http: non-sensical value for parameter: http.buffer.capacity",
			capacity, "using default:", defaultCapacity)
		capacity = defaultCapacity
	}
	buffer := make([]*router.Message, 0, capacity)

	// Determine the buffer timeout
	defaultTimeout, _ := time.ParseDuration("1000ms")
	timeout := getDurationParameter(
		route.Options, "http.buffer.timeout", defaultTimeout)
	timeoutSeconds := timeout.Seconds()
	if timeoutSeconds < .1 || timeoutSeconds > 600 {
		debug("http: non-sensical value for parameter: http.buffer.timeout",
			timeout, "using default:", defaultTimeout)
		timeout = defaultTimeout
	}
	timer := time.NewTimer(timeout)

	// Make the HTTP adapter
	return &HTTPAdapter{
		route:    route,
		url:      url,
		client:   client,
		buffer:   buffer,
		timer:    timer,
		capacity: capacity,
		timeout:  timeout,
	}, nil
}

// Stream implements the router.LogAdapter interface
func (a *HTTPAdapter) Stream(logstream chan *router.Message) {
	for {
		select {
		case message := <-logstream:

			// Append the message to the buffer
			a.bufferMutex.Lock()
			a.buffer = append(a.buffer, message)
			a.bufferMutex.Unlock()

			// Flush if the buffer is at capacity
			if len(a.buffer) >= cap(a.buffer) {
				a.flushHttp("full")
			}
		case <-a.timer.C:

			// Flush if there's anything in the buffer
			if len(a.buffer) > 0 {
				a.flushHttp("timeout")
			}
		}
	}
}

// Flushes the accumulated messages in the buffer
func (a *HTTPAdapter) flushHttp(reason string) {

	// Stop the timer and drain any possible remaining events
	a.timer.Stop()
	select {
	case <-a.timer.C:
	default:
	}

	// Reset the timer when we are done
	defer a.timer.Reset(a.timeout)

	// Capture the buffer and make a new one
	a.bufferMutex.Lock()
	buffer := a.buffer
	a.buffer = make([]*router.Message, 0, a.capacity)
	a.bufferMutex.Unlock()

	// Create JSON representation of all messages
	messages := make([]string, 0, len(buffer))
	for i := range buffer {
		m := buffer[i]
		httpMessage := HTTPMessage{
			Message:  m.Data,
			Time:     m.Time.Format(time.RFC3339),
			Source:   m.Source,
			Name:     m.Container.Name,
			ID:       m.Container.ID,
			Image:    m.Container.Config.Image,
			Hostname: m.Container.Config.Hostname,
		}
		message, err := json.Marshal(httpMessage)
		if err != nil {
			debug("flushHttp - Error encoding JSON: ", err)
			continue
		}
		messages = append(messages, string(message))
	}

	// Glue all the JSON representations together into one payload to send
	payload := strings.Join(messages, "\n")

	go func() {

		// Send the payload.
		request, err := http.NewRequest(
			"POST", a.url, strings.NewReader(payload))
		if err != nil {
			debug("http: error on http.NewRequest:", err, a.url)
			// TODO @raychaser - now what?
			die("", "http: error on http.NewRequest:", err, a.url)
		}
		start := time.Now()
		response, err := a.client.Do(request)
		if err != nil {
			debug("http - error on client.Do:", err, a.url)
			// TODO @raychaser - now what?
			die("http - error on client.Do:", err, a.url)
		}
		if response.StatusCode != 200 {
			debug("http: response not 200 but", response.StatusCode)
			// TODO @raychaser - now what?
			die("http: response not 200 but", response.StatusCode)
		}

		// Make sure the entire response body is read so the HTTP
		// connection can be reused
		io.Copy(ioutil.Discard, response.Body)
		response.Body.Close()

		//		debug(fmt.Sprintf("%#v", request.TLS))
		//		debug(fmt.Sprintf("%#v", response))

		// Bookkeeping, logging
		timeAll := time.Since(start)
		a.totalMessageCount += len(messages)
		debug("http: flushed:", reason, "messages:", len(messages),
			"in:", timeAll, "total:", a.totalMessageCount)
	}()
}

// HTTPMessage is a simple JSON representation of the log message.
type HTTPMessage struct {
	Message  string `json:"message"`
	Time     string `json:"time"`
	Source   string `json:"source"`
	Name     string `json:"docker.name"`
	ID       string `json:"docker.id"`
	Image    string `json:"docker.image"`
	Hostname string `json:"docker.hostname"`
}
