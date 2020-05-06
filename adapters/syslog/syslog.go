package syslog

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"log/syslog"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/gliderlabs/logspout/cfg"
	"github.com/gliderlabs/logspout/router"
)

const (
	// Modern Syslog protocol. https://tools.ietf.org/html/rfc5424
	Rfc5424Format Format = "rfc5424"
	// Legacy BSD Syslog protocol. https://tools.ietf.org/html/rfc3164
	Rfc3164Format Format = "rfc3164"

	// TraditionalTCPFraming is the traditional LF framing of syslog messages on the wire
	TraditionalTCPFraming TCPFraming = "traditional"
	// OctetCountedTCPFraming prepends the size of each message before the message. https://tools.ietf.org/html/rfc6587#section-3.4.1
	OctetCountedTCPFraming TCPFraming = "octet-counted"

	defaultRetryCount = 10
)

var (
	hostname   string
	retryCount uint
	format     Format
	tcpFraming TCPFraming
)

// Format represents the RFC spec to use for syslog messages
type Format string

// TCPFraming represents the type of framing to use for syslog messages
type TCPFraming string

func init() {
	hostname, _ = os.Hostname()
	router.AdapterFactories.Register(NewSyslogAdapter, "syslog")
	setRetryCount()
}

func setRetryCount() {
	if count, err := strconv.Atoi(cfg.GetEnvDefault("RETRY_COUNT", strconv.Itoa(defaultRetryCount))); err != nil {
		retryCount = uint(defaultRetryCount)
	} else {
		retryCount = uint(count)
	}
	debug("setting retryCount to:", retryCount)
}

func debug(v ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		log.Println(v...)
	}
}

func getHostname() string {
	content, err := ioutil.ReadFile("/etc/host_hostname")
	if err == nil && len(content) > 0 {
		hostname = strings.TrimRight(string(content), "\r\n")
	} else {
		hostname = cfg.GetEnvDefault("SYSLOG_HOSTNAME", "{{.Container.Config.Hostname}}")
	}
	return hostname
}

func getFieldTemplates(route *router.Route) (*FieldTemplates, error) {
	var err error
	var s string
	var tmpl FieldTemplates

	s = cfg.GetEnvDefault("SYSLOG_PRIORITY", "{{.Priority}}")
	if tmpl.priority, err = template.New("priority").Parse(s); err != nil {
		return nil, err
	}
	debug("setting priority to:", s)

	s = cfg.GetEnvDefault("SYSLOG_TIMESTAMP", "{{.Timestamp}}")
	if tmpl.timestamp, err = template.New("timestamp").Parse(s); err != nil {
		return nil, err
	}
	debug("setting timestamp to:", s)

	s = getHostname()
	if tmpl.hostname, err = template.New("hostname").Parse(s); err != nil {
		return nil, err
	}
	debug("setting hostname to:", s)

	s = cfg.GetEnvDefault("SYSLOG_TAG", "{{.ContainerName}}"+route.Options["append_tag"])
	if tmpl.tag, err = template.New("tag").Parse(s); err != nil {
		return nil, err
	}
	debug("setting tag to:", s)

	s = cfg.GetEnvDefault("SYSLOG_PID", "{{.Container.State.Pid}}")
	if tmpl.pid, err = template.New("pid").Parse(s); err != nil {
		return nil, err
	}
	debug("setting pid to:", s)

	s = cfg.GetEnvDefault("SYSLOG_STRUCTURED_DATA", "")
	if route.Options["structured_data"] != "" {
		s = route.Options["structured_data"]
	}
	if s == "" {
		s = "-"
	} else {
		s = fmt.Sprintf("[%s]", s)
	}
	if tmpl.structuredData, err = template.New("structuredData").Parse(s); err != nil {
		return nil, err
	}
	debug("setting structuredData to:", s)

	s = cfg.GetEnvDefault("SYSLOG_DATA", "{{.Data}}")
	if tmpl.data, err = template.New("data").Parse(s); err != nil {
		return nil, err
	}
	debug("setting data to:", s)

	return &tmpl, nil
}

// NewSyslogAdapter returnas a configured syslog.Adapter
func NewSyslogAdapter(route *router.Route) (router.LogAdapter, error) {
	transport, found := router.AdapterTransports.Lookup(route.AdapterTransport("udp"))
	if !found {
		return nil, errors.New("bad transport: " + route.Adapter)
	}
	conn, err := transport.Dial(route.Address, route.Options)
	if err != nil {
		return nil, err
	}
	connIsUDP := reflect.TypeOf(conn).String() == "*net.UDPConn"

	if err = setFormat(); err != nil {
		return nil, err
	}
	debug("setting format to:", format)

	var tmpl *FieldTemplates
	if tmpl, err = getFieldTemplates(route); err != nil {
		return nil, err
	}

	if !connIsUDP {
		if err = setTCPFraming(); err != nil {
			return nil, err
		}
		debug("setting tcpFraming to:", tcpFraming)
	}

	return &Adapter{
		route:     route,
		conn:      conn,
		connIsUDP: connIsUDP,
		tmpl:      tmpl,
		transport: transport,
	}, nil
}

// Parses SYSLOG_FORMAT from the environment and sets format
func setFormat() error {
	switch s := cfg.GetEnvDefault("SYSLOG_FORMAT", string(Rfc5424Format)); s {
	case string(Rfc5424Format):
		format = Rfc5424Format
		return nil
	case string(Rfc3164Format):
		format = Rfc3164Format
		return nil
	default:
		return fmt.Errorf("unknown SYSLOG_FORMAT value: %s", s)
	}
}

// Parses SYSLOG_TCP_FRAMING from the environment and sets tcpFraming
func setTCPFraming() error {
	switch s := cfg.GetEnvDefault("SYSLOG_TCP_FRAMING", string(TraditionalTCPFraming)); s {
	case string(TraditionalTCPFraming):
		tcpFraming = TraditionalTCPFraming
		return nil
	case string(OctetCountedTCPFraming):
		tcpFraming = OctetCountedTCPFraming
		return nil
	default:
		return fmt.Errorf("unknown SYSLOG_TCP_FRAMING value: %s", s)
	}
}

// Field templates for rendering Syslog messages
type FieldTemplates struct {
	priority       *template.Template
	timestamp      *template.Template
	hostname       *template.Template
	tag            *template.Template
	pid            *template.Template
	structuredData *template.Template
	data           *template.Template
}

// Adapter streams log output to a connection in the Syslog format
type Adapter struct {
	conn      net.Conn
	connIsUDP bool
	route     *router.Route
	tmpl      *FieldTemplates
	transport router.AdapterTransport
}

// Stream sends log data to a connection
func (a *Adapter) Stream(logstream chan *router.Message) {
	for message := range logstream {
		m := &Message{message}
		buf, err := m.Render(a.tmpl)
		if err != nil {
			log.Println("syslog:", err)
			return
		}

		if !a.connIsUDP && tcpFraming == OctetCountedTCPFraming {
			buf = append([]byte(fmt.Sprintf("%d ", len(buf))), buf...)
		}

		if _, err = a.conn.Write(buf); err != nil {
			log.Println("syslog:", err)
			switch a.conn.(type) {
			case *net.UDPConn:
				continue
			default:
				if err = a.retry(buf, err); err != nil {
					log.Panicf("syslog retry err: %+v", err)
					return
				}
			}
		}
	}
}

func (a *Adapter) retry(buf []byte, err error) error {
	if opError, ok := err.(*net.OpError); ok {
		if (opError.Temporary() && !errors.Is(opError, syscall.ECONNRESET)) || opError.Timeout() {
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

func (a *Adapter) retryTemporary(buf []byte) error {
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

func (a *Adapter) reconnect() error {
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

// Message extends router.Message for the syslog standard
type Message struct {
	*router.Message
}

// Render transforms the log message using the Syslog template
func (m *Message) Render(tmpl *FieldTemplates) ([]byte, error) {
	var err error

	priority := new(bytes.Buffer)
	err = tmpl.priority.Execute(priority, m)
	if err != nil {
		return nil, err
	}

	timestamp := new(bytes.Buffer)
	err = tmpl.timestamp.Execute(timestamp, m)
	if err != nil {
		return nil, err
	}

	hostname := new(bytes.Buffer)
	err = tmpl.hostname.Execute(hostname, m)
	if err != nil {
		return nil, err
	}

	tag := new(bytes.Buffer)
	err = tmpl.tag.Execute(tag, m)
	if err != nil {
		return nil, err
	}

	pid := new(bytes.Buffer)
	err = tmpl.pid.Execute(pid, m)
	if err != nil {
		return nil, err
	}

	structuredData := new(bytes.Buffer)
	err = tmpl.structuredData.Execute(structuredData, m)
	if err != nil {
		return nil, err
	}

	data := new(bytes.Buffer)
	err = tmpl.data.Execute(data, m)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	switch format {
	case Rfc5424Format:
		// notes from RFC:
		// - there is no upper limit for the entire message and depends on the transport in use
		// - the HOSTNAME field must not exceed 255 characters
		// - the TAG field must not exceed 48 characters
		// - the PROCID field must not exceed 128 characters
		fmt.Fprintf(buf, "<%s>1 %s %.255s %.48s %.128s - %s %s\n",
			priority, timestamp, hostname, tag, pid, structuredData, data)
	case Rfc3164Format:
		// notes from RFC:
		// - the entire message must be <= 1024 bytes
		// - the TAG field must not exceed 32 characters
		fmt.Fprintf(buf, "<%s>%s %s %.32s[%s]: %s\n",
			priority, timestamp, hostname, tag, pid, data)
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

// ContainerNameSplitN returns the message's container name sliced at most "n" times using "sep"
func (m *Message) ContainerNameSplitN(sep string, n int) []string {
	return strings.SplitN(m.ContainerName(), sep, n)
}
