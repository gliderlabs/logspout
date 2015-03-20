package router

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RouteManager struct {
	sync.Mutex
	persistor RouteStore
	attacher  LogRouter
	routes    map[string]*Route
}

func NewRouteManager(router LogRouter) *RouteManager {
	return &RouteManager{attacher: router, routes: make(map[string]*Route)}
}

func (rm *RouteManager) Load(persistor RouteStore) error {
	routes, err := persistor.GetAll()
	if err != nil {
		return err
	}
	for _, route := range routes {
		rm.Add(route)
	}
	rm.persistor = persistor
	return nil
}

func (rm *RouteManager) Get(id string) (*Route, error) {
	rm.Lock()
	defer rm.Unlock()
	route, ok := rm.routes[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	return route, nil
}

func (rm *RouteManager) GetAll() ([]*Route, error) {
	rm.Lock()
	defer rm.Unlock()
	routes := make([]*Route, 0)
	for _, route := range rm.routes {
		routes = append(routes, route)
	}
	return routes, nil
}

func (rm *RouteManager) AddFromUri(uri string) error {
	expandedRoute := os.ExpandEnv(uri)
	u, err := url.Parse(expandedRoute)
	if err != nil {
		return err
	}
	r := &Route{
		Address: u.Host,
		Adapter: u.Scheme,
	}
	if u.RawQuery != "" {
		params, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return err
		}
		for key, _ := range params {
			value := params.Get(key)
			switch key {
			case "filter.id":
				r.FilterID = value
			case "filter.name":
				r.FilterName = value
			case "filter.sources":
				r.FilterSources = strings.Split(value, ",")
			default:
				r.Options[key] = value
			}
		}
	}
	return rm.Add(r)
}

func (rm *RouteManager) Add(route *Route) error {
	rm.Lock()
	defer rm.Unlock()
	factory, found := AdapterFactories.Lookup(route.Adapter)
	if !found {
		return errors.New("unable to find adapter: " + route.Adapter)
	}
	adapter, err := factory(route)
	if err != nil {
		return err
	}
	if route.ID == "" {
		h := sha1.New()
		io.WriteString(h, strconv.Itoa(int(time.Now().UnixNano())))
		route.ID = fmt.Sprintf("%x", h.Sum(nil))[:12]
	}
	route.closer = make(chan bool)
	rm.routes[route.ID] = route
	go func() {
		logstream := make(chan *Message)
		defer close(logstream)
		go adapter.Stream(logstream)
		rm.attacher.Route(route, logstream)
	}()
	if rm.persistor != nil {
		if err := rm.persistor.Add(route); err != nil {
			log.Println("persistor:", err)
		}
	}
	return nil
}

func (rm *RouteManager) Remove(id string) bool {
	rm.Lock()
	defer rm.Unlock()
	route, ok := rm.routes[id]
	if ok && route.closer != nil {
		route.closer <- true
	}
	delete(rm.routes, id)
	if rm.persistor != nil {
		rm.persistor.Remove(id)
	}
	return ok
}
