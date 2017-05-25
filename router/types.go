//go:generate go-extpoints . AdapterFactory HttpHandler AdapterTransport LogRouter Job
package router

import (
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
)

// HttpHandler is an extension type for adding HTTP endpoints
type HttpHandler func() http.Handler

// AdapterFactory is an extension type for adding new log adapters
type AdapterFactory func(route *Route) (LogAdapter, error)

// AdapterTransport is an extension type for connection transports used by adapters
type AdapterTransport interface {
	Dial(addr string, options map[string]string) (net.Conn, error)
}

// LogAdapter is a streamed log
type LogAdapter interface {
	Stream(logstream chan *Message)
}

// Job is a thing to be done
type Job interface {
	Run() error
	Setup() error
	Name() string
}

// LogRouter sends logs to LogAdapters via Routes
type LogRouter interface {
	RoutingFrom(containerID string) bool
	Route(route *Route, logstream chan *Message)
}

// RouteStore is a collections of Routes
type RouteStore interface {
	Get(id string) (*Route, error)
	GetAll() ([]*Route, error)
	Add(route *Route) error
	Remove(id string) bool
}

// Message is a log messages
type Message struct {
	Container *docker.Container
	Source    string
	Data      string
	Time      time.Time
}

// Route represents what subset of logs should go where
type Route struct {
	ID            string            `json:"id"`
	FilterID      string            `json:"filter_id,omitempty"`
	FilterName    string            `json:"filter_name,omitempty"`
	FilterSources []string          `json:"filter_sources,omitempty"`
	FilterLabels  []string          `json:"filter_labels,omitempty"`
	Adapter       string            `json:"adapter"`
	Address       string            `json:"address"`
	Options       map[string]string `json:"options,omitempty"`
	adapter       LogAdapter
	closed	      bool
	closer        chan bool
	closerRcv     <-chan bool // used instead of closer when set
}

// AdapterType returns a route's adapter type string
func (r *Route) AdapterType() string {
	return strings.Split(r.Adapter, "+")[0]
}

// AdapterTransport returns a route's adapter transport string
func (r *Route) AdapterTransport(dfault string) string {
	parts := strings.Split(r.Adapter, "+")
	if len(parts) > 1 {
		return parts[1]
	}
	return dfault
}

// Closer returns a route's closerRcv
func (r *Route) Closer() <-chan bool {
	if r.closerRcv != nil {
		return r.closerRcv
	}
	return r.closer
}

// OverrideCloser sets a Route.closer to closer
func (r *Route) OverrideCloser(closer <-chan bool) {
	r.closerRcv = closer
}

// Close sends true to a Route.closer
func (r *Route) Close() {
	r.closer <- true
}

func (r *Route) matchAll() bool {
	if r.FilterID == "" && r.FilterName == "" && len(r.FilterSources) == 0 && len(r.FilterLabels) == 0 {
		return true
	}
	return false
}

// MultiContainer returns whether the Route is matching multiple containers or not
func (r *Route) MultiContainer() bool {
	return r.matchAll() || strings.Contains(r.FilterName, "*")
}

// MatchContainer returns whether the Route is responsible for a given container
func (r *Route) MatchContainer(id, name string, labels map[string]string) bool {
	if r.matchAll() {
		return true
	}
	if r.FilterID != "" && !strings.HasPrefix(id, r.FilterID) {
		return false
	}
	match, err := path.Match(r.FilterName, name)
	if err != nil || (r.FilterName != "" && !match) {
		return false
	}
	for _, label := range r.FilterLabels {
		labelParts := strings.SplitN(label, ":", 2)
		if len(labelParts) > 1 {
			labelKey := labelParts[0]
			labelValue := labelParts[1]
			labelMatch, labelErr := path.Match(labelValue, labels[labelKey])
			if labelErr != nil || (labelValue != "" && !labelMatch) {
				return false
			}
		}
	}

	return true
}

// MatchMessage returns whether the Route is responsible for a given Message
func (r *Route) MatchMessage(message *Message) bool {
	if r.matchAll() {
		return true
	}
	if len(r.FilterSources) > 0 && !contains(r.FilterSources, message.Source) {
		return false
	}
	return true
}

func contains(strs []string, str string) bool {
	for _, s := range strs {
		if s == str {
			return true
		}
	}
	return false
}
