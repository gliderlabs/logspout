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

var address string
var connection net.Conn

func NewRawAdapter(route *router.Route) (router.LogAdapter, error) {
	transport, found := router.AdapterTransports.Lookup(route.AdapterTransport("udp"))
	if !found {
		return nil, errors.New("bad transport: " + route.Adapter)
	}
	address = route.Address
	conn, err := transport.Dial(route.Address, route.Options)
	connection = conn
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
	go connPing(a)
	for message := range logstream {
		buf := new(bytes.Buffer)
		err := a.tmpl.Execute(buf, message)
		if err != nil {
			log.Println("raw:", err)
			return
		}
		//log.Println("debug:", buf.String())

		if cn := utils.M1[message.Container.Name]; cn != "" {
			logmsg := time.Now().Format("2006-01-02T01:47:28.936Z") + " " + utils.UUID + " " + cn + " " + buf.String()
			_, err = connection.Write([]byte(logmsg))
			if err != nil {
                        	//log.Println("raw:", err, reflect.TypeOf(a.conn).String())
                        	if reflect.TypeOf(a.conn).String() != "*net.TCPConn" {
                                	return
                        	}
                	}
		}
	}

}

func connPing() {
	timer := time.NewTicker(2 * time.Second)
        for {
                select {
                case <-timer.C:
			_, err := connection.Write([]byte("ping"))
			if err != nil {
				raddr, err := net.ResolveTCPAddr("tcp", address)
        			if err == nil {
        				conn, err := net.DialTCP("tcp", nil, raddr)
					log.Println("++++", address, err)
        				if err == nil {
						connection = conn
        				}
				}
			}
                }
        }
}
