package rawtcp

import (
	"log"
	"net"
	"os"
	"text/template"

	"github.com/gliderlabs/logspout/router"
)

func init() {
	router.AdapterFactories.Register(NewRawTCPAdapter, "tcp")
}

func NewRawTCPAdapter(route *router.Route) (router.LogAdapter, error) {
	conn, err := net.Dial("tcp", route.Address)
	if err != nil {
		return nil, err
	}
	tmplStr := "{{.Data}}\n"
	if os.Getenv("RAWTCP_TEMPLATE") != "" {
		tmplStr = os.Getenv("RAWTCP_TEMPLATE")
	}
	tmpl, err := template.New("tcp").Parse(tmplStr)
	return &RawTCPAdapter{
		route: route,
		conn:  conn,
		tmpl:  tmpl,
	}, nil
}

type RawTCPAdapter struct {
	conn  net.Conn
	route *router.Route
	tmpl  *template.Template
}

func (a *RawTCPAdapter) Stream(logstream chan *router.Message) {
	for message := range logstream {
		if !a.route.Match(message) {
			continue
		}
		err := a.tmpl.Execute(a.conn, message)
		if err != nil && os.Getenv("DEBUG") != "" {
			log.Println("rawtcp:", err)
			a.route.Close()
			return
		}
	}
}
