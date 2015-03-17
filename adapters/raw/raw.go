package raw

import (
	"errors"
	"log"
	"net"
	"os"
	"text/template"

	"github.com/gliderlabs/logspout/router"
)

func init() {
	router.AdapterFactories.Register(NewRawAdapter, "raw")
}

func NewRawAdapter(route *router.Route) (router.LogAdapter, error) {
	connFactory, found := router.ConnectionFactories.Lookup(route.AdapterConnType("udp"))
	if !found {
		return nil, errors.New("unable to find adapter: " + route.Adapter)
	}
	conn, err := connFactory(route.Address, route.Options)
	if err != nil {
		return nil, err
	}
	tmplStr := "{{.Data}}\n"
	if os.Getenv("RAW_FORMAT") != "" {
		tmplStr = os.Getenv("RAW_FORMAT")
	}
	tmpl, err := template.New("raw").Parse(tmplStr)
	if err != nil {
		return nil, err
	}
	return &RawAdapter{
		route: route,
		conn:  conn,
		tmpl:  tmpl,
	}, nil
}

type RawAdapter struct {
	conn  net.Conn
	route *router.Route
	tmpl  *template.Template
}

func (a *RawAdapter) Stream(logstream chan *router.Message) {
	for message := range logstream {
		err := a.tmpl.Execute(a.conn, message)
		if err != nil {
			log.Println("raw:", err)
			a.route.Close()
			return
		}
	}
}
