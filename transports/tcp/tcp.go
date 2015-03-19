package tcp

import (
	"net"

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
	raddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
