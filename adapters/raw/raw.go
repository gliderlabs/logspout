package raw

import (
	"bytes"
	"errors"
	"log"
	"net"
	"os"
	"reflect"
	"text/template"

	"github.com/gliderlabs/logspout/router"
	"time"
)

func init() {
	router.AdapterFactories.Register(NewRawAdapter, "raw")
}

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
		buf := new(bytes.Buffer)
		err := a.tmpl.Execute(buf, message)
		if err != nil {
			log.Println("raw:", err)
			return
		}
		//log.Println("debug:", buf.String())
		logmsg := time.Now().Format("2006-01-02T01:47:28.936Z") + " " + router.UUID + " " + message.Container.Name + " " + buf.String()
		//_, err = a.conn.Write(buf.Bytes())
		_, err = a.conn.Write([]byte(logmsg))
		if err != nil {
                        log.Println("raw:", err)
                        if reflect.TypeOf(a.conn).String() != "*net.UDPConn" {
                                return
                        }
                }
	}

}
