package logstash

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"

	"github.com/gliderlabs/logspout/router"
)

var hostname string

func init() {
	router.AdapterFactories.Register(NewLogstashAdapter, "logstash")

	hostname, _ = os.Hostname()
}

// LogstashAdapter is an adapter that streams UPD JSON to Logstash.
type LogstashAdapter struct {
	conn  net.Conn
	route *router.Route
}

// NewLogstashAdapter creates a LogstashAdapter with UDP transport.
func NewLogstashAdapter(route *router.Route) (router.LogAdapter, error) {
	transport, found := router.AdapterTransports.Lookup(route.AdapterTransport("udp"))
	if !found {
		return nil, errors.New("unable to find adapter: " + route.Adapter)
	}

	conn, err := transport.Dial(route.Address, route.Options)
	if err != nil {
		return nil, err
	}

	return &LogstashAdapter{
		route: route,
		conn:  conn,
	}, nil
}

func (adapter *LogstashAdapter) Stream(logstream chan *router.Message) {
	for m := range logstream {
		msg := LogstashMessage{
			Time:     m.Time.Unix(),
			Message:  m.Data,
			Hostname: hostname,
			Image:    m.Container.Config.Image,
		}
		js, _ := json.Marshal(msg)
		io.WriteString(adapter.conn, string(js))
	}
}

type LogstashMessage struct {
	Time     int64  `json:"time"`
	Message  string `json:"message"`
	Hostname string `json:"hostname"`
	Image    string `json:"image"`
}
