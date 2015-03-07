//go:generate go-extpoints . AdapterFactory HttpHandler
package router

import (
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
)

type HttpHandler func(routes *RouteManager, router LogRouter) http.Handler

type AdapterFactory func(route *Route) (LogAdapter, error)

type LogAdapter interface {
	Stream(logstream chan *Message)
}

type LogRouter interface {
	Routing(containerID string) bool
	Route(route *Route, logstream chan *Message)
}

type RouteStore interface {
	Get(id string) (*Route, error)
	GetAll() ([]*Route, error)
	Add(route *Route) error
	Remove(id string) bool
}

type Message struct {
	Container *docker.Container
	Source    string
	Data      string
	Time      time.Time
}

type Route struct {
	ID            string            `json:"id"`
	FilterID      string            `json:"filter_id,omitempty"`
	FilterName    string            `json:"filter_name,omitempty"`
	FilterSources []string          `json:"filter_sources,omitempty"`
	Adapter       string            `json:"adapter"`
	Address       string            `json:"address"`
	Options       map[string]string `json:"options,omitempty"`
	closer        chan bool
	closerRcv     <-chan bool
}

func (r *Route) Closer() <-chan bool {
	if r.closerRcv != nil {
		return r.closerRcv
	}
	return r.closer
}

func (r *Route) OverrideCloser(closer <-chan bool) {
	r.closerRcv = closer
}

func (r *Route) Close() {
	r.closer <- true
}

func (r *Route) MultiContainer() bool {
	return r.MatchAll() || strings.Contains(r.FilterName, "*")
}

func (r *Route) MatchAll() bool {
	if r.FilterID == "" && r.FilterName == "" && len(r.FilterSources) == 0 {
		return true
	}
	return false
}

func (r *Route) MatchContainer(id, name string) bool {
	if r.FilterID != "" && !strings.HasPrefix(id, r.FilterID) {
		return false
	}
	match, err := path.Match(r.FilterName, name)
	if err != nil || (r.FilterName != "" && match) {
		return false
	}
	return true
}

func (r *Route) Match(message *Message) bool {
	if r.MatchAll() {
		return true
	}
	if len(r.FilterSources) > 0 && !contains(r.FilterSources, message.Source) {
		return false
	}
	return r.MatchContainer(message.Container.ID, message.Container.Name[1:])
}

func contains(strs []string, str string) bool {
	for _, s := range strs {
		if s == str {
			return true
		}
	}
	return false
}
