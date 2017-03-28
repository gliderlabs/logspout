package tcp

import (
	"net"
	"time"

	"github.com/gliderlabs/logspout/adapters/raw"
	"github.com/gliderlabs/logspout/router"
)

func init() {
	router.AdapterTransports.Register(new(tcpTransport), "tcp")
	// convenience adapters around raw adapter
	router.AdapterFactories.Register(rawTCPAdapter, "tcp")
}

func rawTCPAdapter(route *router.Route) (router.LogAdapter, error) {
	route.Adapter = "raw+tcp"
	return raw.NewRawAdapter(route)
}

type tcpTransport int

func (_ *tcpTransport) Dial(addr string, options map[string]string) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
