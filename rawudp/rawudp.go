package rawudp

import (
	"log"
	"net"
	"os"
	"text/template"

	"github.com/gliderlabs/logspout/router"
)

const (
	writeBuffer = 1024 * 1024
)

func init() {
	router.AdapterFactories.Register(NewRawUDPAdapter, "udp")
}

func NewRawUDPAdapter(route *router.Route) (router.LogAdapter, error) {
	addr, err := net.ResolveUDPAddr("udp", route.Address)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}
	// bump up the packet size for large log lines
	err = conn.SetWriteBuffer(writeBuffer)
	if err != nil {
		return nil, err
	}
	tmplStr := "{{.Data}}\n"
	if os.Getenv("RAWUDP_TEMPLATE") != "" {
		tmplStr = os.Getenv("RAWUDP_TEMPLATE")
	}
	tmpl, err := template.New("rawudp").Parse(tmplStr)
	if err != nil {
		return nil, err
	}
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
		err := a.tmpl.Execute(a.conn, message)
		if err != nil {
			log.Println("rawudp:", err)
			a.route.Close()
			return
		}
	}
}
