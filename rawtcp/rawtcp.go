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
	addr, err := net.ResolveTCPAddr("tcp", route.Address)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}
	tmplStr := "{{.Data}}\n"
	if os.Getenv("RAWTCP_TEMPLATE") != "" {
		tmplStr = os.Getenv("RAWTCP_TEMPLATE")
	}
	tmpl, err := template.New("rawtcp").Parse(tmplStr)
	if err != nil {
		return nil, err
	}
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
		err := a.tmpl.Execute(a.conn, message)
		if err != nil {
			log.Println("rawtcp:", err)
			a.route.Close()
			return
		}
	}
}
