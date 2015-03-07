package rawudp

import (
	"log"
	"net"
	"os"
	"text/template"

	"github.com/gliderlabs/logspout/router"
)

func init() {
	router.AdapterFactories.Register(NewRawUDPAdapter, "udp")
}

func NewRawUDPAdapter(route *router.Route) (router.LogAdapter, error) {
	conn, err := net.Dial("udp", route.Address)
	if err != nil {
		return nil, err
	}
	tmplStr := "{{.Data}}\n"
	if os.Getenv("RAWUDP_TEMPLATE") != "" {
		tmplStr = os.Getenv("RAWUDP_TEMPLATE")
	}
	tmpl, err := template.New("udp").Parse(tmplStr)
	return &RawUDPAdapter{
		route: route,
		conn:  conn,
		tmpl:  tmpl,
	}, nil
}

type RawUDPAdapter struct {
	conn  net.Conn
	route *router.Route
	tmpl  *template.Template
}

func (a *RawUDPAdapter) Stream(logstream chan *router.Message) {
	for message := range logstream {
		if !a.route.Match(message) {
			continue
		}
		err := a.tmpl.Execute(a.conn, message)
		if err != nil && os.Getenv("DEBUG") != "" {
			log.Println("rawudp:", err)
			a.route.Close()
			return
		}
	}
}
