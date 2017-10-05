package tcp

import (
	"net"

	"github.com/gliderlabs/logspout/adapters/raw"
	"github.com/gliderlabs/logspout/router"
	"github.com/streadway/amqp"
)

func init() {
	router.AdapterTransports.Register(new(amqpTransport), "amqp")
	// convenience adapters around raw adapter
	router.AdapterFactories.Register(rawAMQPAdapter, "amqp")
}

func rawAMQPAdapter(route *router.Route) (router.LogAdapter, error) {
	route.Adapter = "raw+amqp"
	return raw.NewRawAdapter(route)
}

type amqpTransport int

func (_ *amqpTransport) Dial(addr string, options map[string]string) (net.Conn, error) {
	connection, err := amqp.Dial(addr)
	if err != nil {
		return nil, err
	}
	return connection, nil
}
