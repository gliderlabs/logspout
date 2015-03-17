package udp

import (
	"net"

	"github.com/gliderlabs/logspout/router"
)

const (
	// make configurable?
	writeBuffer = 1024 * 1024
)

func init() {
	router.ConnectionFactories.Register(NewUDPFactory, "udp")
	// convenience adapters around raw adapter
	router.AdapterFactories.Register(NewRawUDPAdapter, "udp")
}

func NewRawUDPAdapter(route *router.Route) (router.LogAdapter, error) {
	route.Adapter = "raw+udp"
	return raw.NewRawAdapter(route)
}

func NewUDPFactory(addr string, options map[string]string) (net.Conn, error) {
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, err
	}
	// bump up the packet size for large log lines
	err = conn.SetWriteBuffer(writeBuffer)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
