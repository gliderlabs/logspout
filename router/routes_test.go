package router

import (
	"reflect"
	"testing"
)

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
