package tls

import (
	"net"
	"crypto/tls"

	"github.com/gliderlabs/logspout/adapters/raw"
	"github.com/gliderlabs/logspout/router"
)

func init() {
	router.AdapterTransports.Register(new(tlsTransport), "tls")
	// convenience adapters around raw adapter
	router.AdapterFactories.Register(rawTLSAdapter, "tls")
}

func rawTLSAdapter(route *router.Route) (router.LogAdapter, error) {
	route.Adapter = "raw+tls"
	return raw.NewRawAdapter(route)
}

type tlsTransport int

func (_ *tlsTransport) Dial(addr string, options map[string]string) (net.Conn, error) {
	conn, err := tls.Dial("tcp",  addr, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
