package raw

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net"
	"os"
	"text/template"

	"github.com/gliderlabs/logspout/router"
	"strconv"
	"time"
)

const defaultRetryCount = 10

var retryCount uint

func init() {
	router.AdapterFactories.Register(NewRawAdapter, "raw")
	setRetryCount()
}

func setRetryCount() {
	if count, err := strconv.Atoi(getopt("RETRY_COUNT", strconv.Itoa(defaultRetryCount))); err != nil {
		retryCount = uint(defaultRetryCount)
	} else {
		retryCount = uint(count)
	}
	debug("setting retryCount to:", retryCount)
}

func getopt(name, dfault string) string {
	value := os.Getenv(name)
	if value == "" {
		value = dfault
	}
	return value
}

func debug(v ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		log.Println(v...)
	}
}

var funcs = template.FuncMap{
	"toJSON": func(value interface{}) string {
		bytes, err := json.Marshal(value)
		if err != nil {
			log.Println("error marshalling to JSON: ", err)
			return "null"
		}
		return string(bytes)
	},
}

// NewRawAdapter returns a configured raw.Adapter
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
	tmpl, err := template.New("raw").Funcs(funcs).Parse(tmplStr)
	if err != nil {
		return nil, err
	}
	return &Adapter{
		route:     route,
		conn:      conn,
		tmpl:      tmpl,
		transport: transport,
	}, nil
}

// Adapter is a simple adapter that streams log output to a connection without any templating
type Adapter struct {
	conn      net.Conn
	route     *router.Route
	tmpl      *template.Template
	transport router.AdapterTransport
}

// Stream sends log data to a connection
func (a *Adapter) Stream(logstream chan *router.Message) {
	for message := range logstream {
		buf := new(bytes.Buffer)
		err := a.tmpl.Execute(buf, message)
		if err != nil {
			log.Println("raw:", err)
			return
		}
		//log.Println("debug:", buf.String())
		_, err = a.conn.Write(buf.Bytes())
		if err != nil {
			log.Println("raw:", err)
			switch a.conn.(type) {
			case *net.UDPConn:
				continue
			default:
				if err = a.retry(buf.Bytes(), err); err != nil {
					log.Panicf("raw retry err: %+v", err)
					return
				}
			}
		}
	}
}

func (a *Adapter) retry(buf []byte, err error) error {
	if opError, ok := err.(*net.OpError); ok {
		if opError.Temporary() || opError.Timeout() {
			retryErr := a.retryTemporary(buf)
			if retryErr == nil {
				return nil
			}
		}
	}
	if reconnErr := a.reconnect(); reconnErr != nil {
		return reconnErr
	}
	if _, err = a.conn.Write(buf); err != nil {
		log.Println("raw: reconnect failed")
		return err
	}
	log.Println("raw: reconnect successful")
	return nil
}

func (a *Adapter) retryTemporary(buf []byte) error {
	log.Printf("raw: retrying tcp up to %v times\n", retryCount)
	err := retryExp(func() error {
		_, err := a.conn.Write(buf)
		if err == nil {
			log.Println("raw: retry successful")
			return nil
		}

		return err
	}, retryCount)

	if err != nil {
		log.Println("raw: retry failed")
		return err
	}

	return nil
}

func (a *Adapter) reconnect() error {
	log.Printf("raw: reconnecting up to %v times\n", retryCount)
	err := retryExp(func() error {
		conn, err := a.transport.Dial(a.route.Address, a.route.Options)
		if err != nil {
			return err
		}
		a.conn = conn
		return nil
	}, retryCount)

	if err != nil {
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

