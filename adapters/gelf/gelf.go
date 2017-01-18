package gelf

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/gliderlabs/logspout/router"
)

// http://docs.graylog.org/en/2.1/pages/gelf.html

var hostname string

func init() {
	hostname, _ = os.Hostname()
	router.AdapterFactories.Register(NewGelfAdapter, "gelf")
}

func NewGelfAdapter(route *router.Route) (router.LogAdapter, error) {
	transport, found := router.AdapterTransports.Lookup(route.AdapterTransport("udp"))
	if !found {
		return nil, fmt.Errorf("bad transport: %s", route.Adapter)
	}
	conn, err := transport.Dial(route.Address, route.Options)
	if err != nil {
		return nil, err
	}

	return &GelfAdapter{
		conn:      conn,
		route:     route,
		transport: transport,
	}, nil
}

type GelfAdapter struct {
	conn      net.Conn
	route     *router.Route
	transport router.AdapterTransport
}

func (a *GelfAdapter) Stream(logstream chan *router.Message) {
	for m := range logstream {
		msg := GelfMessage{
			Version:            "1.1",
			Host:               hostname,
			ShortMessage:       m.Data,
			Timestamp:          float64(m.Time.UnixNano()) / float64(time.Second),
			ContainerID:        m.Container.ID,
			ContainerName:      m.Container.Name[1:],
			ContainerImageID:   m.Container.Image,
			ContainerImageName: m.Container.Config.Image,
		}
		buf, err := json.Marshal(msg)
		if err != nil {
			log.Println("gelf:", err)
			continue
		}
		_, err = a.conn.Write(buf)
		if err != nil {
			log.Println("gelf:", err)
			switch a.conn.(type) {
			case *net.UDPConn:
				continue
			default:
				err = a.retry(buf, err)
				if err != nil {
					log.Println("gelf:", err)
					return
				}
			}
		}
	}
}

func (a *GelfAdapter) retry(buf []byte, err error) error {
	if opError, ok := err.(*net.OpError); ok {
		if opError.Temporary() || opError.Timeout() {
			retryErr := a.retryTemporary(buf)
			if retryErr == nil {
				return nil
			}
		}
	}

	return a.reconnect()
}

func (a *GelfAdapter) retryTemporary(buf []byte) error {
	log.Println("gelf: retrying udp up to 11 times")
	err := retryExp(func() error {
		_, err := a.conn.Write(buf)
		if err == nil {
			log.Println("gelf: retry successful")
			return nil
		}

		return err
	}, 11)

	if err != nil {
		log.Println("gelf: retry failed")
		return err
	}

	return nil
}

func (a *GelfAdapter) reconnect() error {
	log.Println("gelf: reconnecting up to 11 times")
	err := retryExp(func() error {
		conn, err := a.transport.Dial(a.route.Address, a.route.Options)
		if err != nil {
			return err
		}

		a.conn = conn
		return nil
	}, 11)

	if err != nil {
		log.Println("gelf: reconnect failed")
		return err
	}

	return nil
}

func retryExp(fun func() error, tries uint) error {
	try := uint(0)
	for {
		err := fun()
		if err == nil {
			return nil
		}

		try++
		if try > tries {
			return err
		}

		time.Sleep((1 << try) * 10 * time.Millisecond)
	}
}

// GelfMessage GELF Version 1.1 (11/2013)
// http://docs.graylog.org/en/2.1/pages/gelf.html#gelf-format-specification
type GelfMessage struct {
	Version      string  `json:"version"`
	Host         string  `json:"host"`
	ShortMessage string  `json:"short_message"`
	FullMessage  string  `json:"full_message,omitempty"`
	Timestamp    float64 `json:"timestamp,omitempty"`
	Level        int     `json:"level,omitempty"`
	// Additional fields
	ContainerID        string `json:"_docker_container_id,omitempty"`
	ContainerName      string `json:"_docker_container_name,omitempty"`
	ContainerImageID   string `json:"_docker_image_id,omitempty"`
	ContainerImageName string `json:"_docker_image_name,omitempty"`
}
