package raw

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net"
	"os"
	"reflect"
	"text/template"

	"github.com/gliderlabs/logspout/router"
)

func init() {
	router.AdapterFactories.Register(NewRawAdapter, "raw")
}

var funcs = template.FuncMap{
	"toJSON": func(value interface{}) string {
		bytes, err := json.Marshal(value)
		if err != nil {
			log.Println("error marshalling to JSON: ", err)
			return "null"
		}
		return string(bytes)
	},
}

// NewRawAdapter returns a configured raw.Adapter
func NewRawAdapter(route *router.Route) (router.LogAdapter, error) {
	transport, found := router.AdapterTransports.Lookup(route.AdapterTransport("udp"))
	if !found {
		return nil, errors.New("bad transport: " + route.Adapter)
	}
	conn, err := transport.Dial(route.Address, route.Options)
	if err != nil {
		return nil, err
	}
	tmplStr := "{{.Data}}\n"
	if os.Getenv("RAW_FORMAT") != "" {
		tmplStr = os.Getenv("RAW_FORMAT")
	}
	tmpl, err := template.New("raw").Funcs(funcs).Parse(tmplStr)
	if err != nil {
		return nil, err
	}
	return &Adapter{
		route: route,
		conn:  conn,
		tmpl:  tmpl,
	}, nil
}

// Adapter is a simple adapter that streams log output to a connection without any templating
type Adapter struct {
	conn  net.Conn
	route *router.Route
	tmpl  *template.Template
}

// Stream sends log data to a connection
func (a *Adapter) Stream(logstream chan *router.Message) {
	for message := range logstream {
		buf := new(bytes.Buffer)
		err := a.tmpl.Execute(buf, message)
		if err != nil {
			log.Println("raw:", err)
			return
		}
		//log.Println("debug:", buf.String())
		_, err = a.conn.Write(buf.Bytes())
		if err != nil {
			log.Println("raw:", err)
			if reflect.TypeOf(a.conn).String() != "*net.UDPConn" {
				return
			}
		}
	}
}
