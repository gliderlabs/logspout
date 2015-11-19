package kafka

import (
	"net"

	"github.com/gliderlabs/logspout/adapters/kafkaraw"
	"github.com/gliderlabs/logspout/router"
)

func init() {
	router.AdapterTransports.Register(new(kafkaTransport), "kafka")
	// convenience adapters around raw adapter
	router.AdapterFactories.Register(rawKafkaAdapter, "kafka")
}

func rawKafkaAdapter(route *router.Route) (router.LogAdapter, error) {
	route.Adapter = "kafkaraw+tcp"
	return kafkaraw.NewKafkaRawAdapter(route)
}

type kafkaTransport int

func (_ *kafkaTransport) Dial(addr string, options map[string]string) (net.Conn, error) {
	/*producer, err := kafka.NewSyncProducer([]string{addr}, nil)
	if err != nil {
		return nil, err
	}*/
	return nil, nil
}
