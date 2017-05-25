package router

import (
	"reflect"
	"testing"
)

type DummyAdapter struct{}

func (a *DummyAdapter) Stream(logstream chan *Message) {
	for message := range logstream {
		print("message passed to DummyAdapter: ", message)
	}
}

func newDummyAdapter(route *Route) (LogAdapter, error) {
	return &DummyAdapter{}, nil
}

func TestRouterGetAll(t *testing.T) {
	rts, err := Routes.GetAll()
	if err != nil {
		t.Error("error getting all routes")
	}
	routes := append(marshal(rts), '\n')
	emptyRoutes := append(marshal(make([]*Route, 0)), '\n')
	if !reflect.DeepEqual(routes, emptyRoutes) {
		t.Error("expected '[]' got:", routes)
	}
}

func TestRouterNoDuplicateIds(t *testing.T) {
	AdapterFactories.Register(newDummyAdapter, "syslog")

	//Mock "running" so routes actually start running when added.
	Routes.routing = true

	//Start the first route.
	route1 := &Route{
		ID:      "abc",
		Address: "someUrl",
		Adapter: "syslog",
	}
	if err := Routes.Add(route1); err != nil {
		t.Error("Error adding route:", err)
	}

	//Start a second route with the same ID.
	var route2 = &Route{
		ID:      "abc",
		Address: "someUrl2",
		Adapter: "syslog",
	}
	Routes.Add(route2)

	if !route1.closed {
		t.Errorf("route1 was not closed after route2 added.")
	}
}
