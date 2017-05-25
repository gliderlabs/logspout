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

// Routes is all the configured routes
var Routes *RouteManager

func init() {
	Routes = &RouteManager{routes: make(map[string]*Route)}
	Jobs.Register(Routes, "routes")
}

// RouteManager is responsible for maintaining route state
type RouteManager struct {
	sync.Mutex
	persistor RouteStore
	routes    map[string]*Route
	routing   bool
	wg        sync.WaitGroup
}

// Load loads all route from a RouteStore
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

// Get returns a Route based on id
func (rm *RouteManager) Get(id string) (*Route, error) {
	rm.Lock()
	defer rm.Unlock()
	route, ok := rm.routes[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	return route, nil
}

// GetAll returns all routes in the RouteManager
func (rm *RouteManager) GetAll() ([]*Route, error) {
	rm.Lock()
	defer rm.Unlock()
	routes := make([]*Route, 0)
	for _, route := range rm.routes {
		routes = append(routes, route)
	}
	return routes, nil
}

// Remove removes a route from a RouteManager based on id
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

// AddFromURI creates a new route from an URI string and adds it to the RouteManager
func (rm *RouteManager) AddFromURI(uri string) error {
	expandedRoute := os.ExpandEnv(uri)
	u, err := url.Parse(expandedRoute)
	if err != nil {
		return err
	}
	r := &Route{
		Address: u.Host,
		Adapter: u.Scheme,
		Options: make(map[string]string),
	}
	if u.RawQuery != "" {
		params, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return err
		}
		for key := range params {
			value := params.Get(key)
			switch key {
			case "filter.id":
				r.FilterID = value
			case "filter.name":
				r.FilterName = value
			case "filter.labels":
				r.FilterLabels = strings.Split(value, ",")
			case "filter.sources":
				r.FilterSources = strings.Split(value, ",")
			default:
				r.Options[key] = value
			}
		}
	}
	return rm.Add(r)
}

// Add adds a route to the RouteManager
func (rm *RouteManager) Add(route *Route) error {
	rm.Lock()
	defer rm.Unlock()
	factory, found := AdapterFactories.Lookup(route.AdapterType())
	if !found {
		return errors.New("bad adapter: " + route.Adapter)
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
	route.adapter = adapter
	//Stop any existing route with this ID:
	if rm.routes[route.ID] != nil {
		rm.routes[route.ID].closer <- true
	}

	rm.routes[route.ID] = route
	if rm.persistor != nil {
		if err := rm.persistor.Add(route); err != nil {
			log.Println("persistor:", err)
		}
	}
	if rm.routing {
		go rm.route(route)
	}
	return nil
}

func (rm *RouteManager) route(route *Route) {
	logstream := make(chan *Message)
	defer route.Close()
	rm.Route(route, logstream)
	route.adapter.Stream(logstream)
}

// Route takes a logstream and route and passes them off to all configure LogRouters
func (rm *RouteManager) Route(route *Route, logstream chan *Message) {
	for _, router := range LogRouters.All() {
		go router.Route(route, logstream)
	}
}

// RoutingFrom returns whether a given container is routing through the RouteManager
func (rm *RouteManager) RoutingFrom(containerID string) bool {
	for _, router := range LogRouters.All() {
		if router.RoutingFrom(containerID) {
			return true
		}
	}
	return false
}

// Run executes the RouteManager
func (rm *RouteManager) Run() error {
	rm.Lock()
	for _, route := range rm.routes {
		rm.wg.Add(1)
		go func(route *Route) {
			rm.route(route)
			rm.wg.Done()
		}(route)
	}
	rm.routing = true
	rm.Unlock()
	rm.wg.Wait()
	// Temp fix to allow logspout to run without routes defined.
	if len(rm.routes) == 0 {
		select {}
	}
	return nil
}

// Name returns the name of the RouteManager
func (rm *RouteManager) Name() string {
	return "routes"
}

// Setup configures the RouteManager
func (rm *RouteManager) Setup() error {
	var uris string
	if os.Getenv("ROUTE_URIS") != "" {
		uris = os.Getenv("ROUTE_URIS")
	}
	if len(os.Args) > 1 {
		uris = os.Args[1]
	}
	if uris != "" {
		for _, uri := range strings.Split(uris, ",") {
			err := rm.AddFromURI(uri)
			if err != nil {
				return err
			}
		}
	}

	persistPath := getopt("ROUTESPATH", "/mnt/routes")
	if _, err := os.Stat(persistPath); err == nil {
		return rm.Load(RouteFileStore(persistPath))
	}
	return nil
}
