package syslogamqp

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"log/syslog"

	"os"
	"strconv"
	"syscall"
	"text/template"
	"time"

	"github.com/gliderlabs/logspout/router"
	"github.com/streadway/amqp"
)

const defaultRetryCount = 10

var (
	hostname         string
	retryCount       uint
	econnResetErrStr string
)

func init() {
	hostname, _ = os.Hostname()
	econnResetErrStr = fmt.Sprintf("write: %s", syscall.ECONNRESET.Error())
	router.AdapterFactories.Register(NewSyslogAMQPAdapter, "syslogamqp")
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

// NewSyslogAMQPAdapter returnas a configured syslog.Adapter
func NewSyslogAMQPAdapter(route *router.Route) (router.LogAdapter, error) {
	//uri := "amqp://" + a.user + ":" + a.password + "@" + a.address
	connection, err := amqp.Dial(route.Address)
	if err != nil {
		log.Printf("amqp.Dial: %s - " + route.Address, err)
		return nil, err
	}
	channel, err := connection.Channel()
	if err != nil {
			log.Printf("connection.Channel(): %s", err)
		return nil, err
	}

	format := getopt("SYSLOG_FORMAT", "rfc5424")
	priority := getopt("SYSLOG_PRIORITY", "{{.Priority}}")
	pid := getopt("SYSLOG_PID", "{{.Container.State.Pid}}")

	content, err := ioutil.ReadFile("/etc/host_hostname") // just pass the file name
	if err == nil && len(content) > 0{
		hostname = string(content) // convert content to a 'string'
	} else {
		hostname = getopt("SYSLOG_HOSTNAME", "{{.Container.Config.Hostname}}")
	}
	tag := getopt("SYSLOG_TAG", "{{.ContainerName}}"+route.Options["append_tag"])
	structuredData := getopt("SYSLOG_STRUCTURED_DATA", "")
	if route.Options["structured_data"] != "" {
		structuredData = route.Options["structured_data"]
	}
	data := getopt("SYSLOG_DATA", "{{.Data}}")
	timestamp := getopt("SYSLOG_TIMESTAMP", "{{.Timestamp}}")

	if structuredData == "" {
		structuredData = "-"
	} else {
		structuredData = fmt.Sprintf("[%s]", structuredData)
	}

	var tmplStr string
	switch format {
	case "rfc5424":
		tmplStr = fmt.Sprintf("<%s>1 %s %s %s %s - %s %s\n",
			priority, timestamp, hostname, tag, pid, structuredData, data)
	case "rfc3164":
		tmplStr = fmt.Sprintf("<%s>%s %s %s[%s]: %s\n",
			priority, timestamp, hostname, tag, pid, data)
	default:
		return nil, errors.New("unsupported syslog format: " + format)
	}
	tmpl, err := template.New("syslog").Parse(tmplStr)
	if err != nil {
		return nil, err
	}
	return &AMQPAdapter {
		route:      route,
		//connection: connection,
		channel: 	  channel,
		exchange:   getopt("AMQP_EXCHANGE", "logspout"),
		routingKey: getopt("AMQP_ROUTING_KEY", "docker"),
		tmpl:       tmpl,
		//transport:  transport,
	}, nil
}

// Adapter publishes log output to an AMQP exchange in the Syslog format
type AMQPAdapter struct {
	//connection      amqp.Connection
	channel         *amqp.Channel
	exchange				string
	routingKey			string
	route           *router.Route
	tmpl            *template.Template
	//transport       router.AdapterTransport
}

// Stream sends log data to a connection
func (a *AMQPAdapter) Stream(logstream chan *router.Message) {
	for message := range logstream {
		m := &Message{message}
		buf, err := m.Render(a.tmpl)
		if err != nil {
			log.Println("syslog:", err)
			return
		}

		amqpMessage := amqp.Publishing{
			//Headers: amqp.Table{},
			//ContentType: "text/plain",
			//ContentEncoding: "UTF-8",
			//DeliveryMode: amqp.Transient,
			DeliveryMode: amqp.Persistent,
			Priority: 0,
			Timestamp:    time.Now(),
			Body:         buf,
		}

		err = a.channel.Publish(
			a.exchange, // exchange
			a.routingKey, // routing key
			false, // mandatory
		  false, //immediate
		  amqpMessage,
		)

		if err != nil {
			log.Println("syslog:", err)
		}
	}
}
/*
func (a *AMQPAdapter) retry(buf []byte, err error) error {
	if opError, ok := err.(*net.OpError); ok {
		if (opError.Temporary() && opError.Err.Error() != econnResetErrStr) || opError.Timeout() {
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
		log.Println("syslog: reconnect failed")
		return err
	}
	log.Println("syslog: reconnect successful")
	return nil
}

func (a *AMQPAdapter) retryTemporary(buf []byte) error {
	log.Printf("syslog: retrying tcp up to %v times\n", retryCount)
	err := retryExp(func() error {
		_, err := a.conn.Write(buf)
		if err == nil {
			log.Println("syslog: retry successful")
			return nil
		}

		return err
	}, retryCount)

	if err != nil {
		log.Println("syslog: retry failed")
		return err
	}

	return nil
}

func (a *AMQPAdapter) reconnect() error {
	log.Printf("syslog: reconnecting up to %v times\n", retryCount)
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
*/
// Message extends router.Message for the syslog standard
type Message struct {
	*router.Message
}

// Render transforms the log message using the Syslog template
func (m *Message) Render(tmpl *template.Template) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := tmpl.Execute(buf, m)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Priority returns a syslog.Priority based on the message source
func (m *Message) Priority() syslog.Priority {
	switch m.Message.Source {
	case "stdout":
		return syslog.LOG_USER | syslog.LOG_INFO
	case "stderr":
		return syslog.LOG_USER | syslog.LOG_ERR
	default:
		return syslog.LOG_DAEMON | syslog.LOG_INFO
	}
}

// Hostname returns the os hostname
func (m *Message) Hostname() string {
	return hostname
}

// Timestamp returns the message's syslog formatted timestamp
func (m *Message) Timestamp() string {
	return m.Message.Time.Format(time.RFC3339)
}

// ContainerName returns the message's container name
func (m *Message) ContainerName() string {
	return m.Message.Container.Name[1:]
}
