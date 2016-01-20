package syslog

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"log/syslog"
	"net"
	"os"
	"reflect"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/gliderlabs/logspout/router"
)

var hostname string

func init() {
	hostname, _ = os.Hostname()
	router.AdapterFactories.Register(NewSyslogAdapter, "syslog")
}

func getopt(name, dfault string) string {
	value := os.Getenv(name)
	if value == "" {
		value = dfault
	}
	return value
}

func NewSyslogAdapter(route *router.Route) (router.LogAdapter, error) {
	transport, found := router.AdapterTransports.Lookup(route.AdapterTransport("udp"))
	if !found {
		return nil, errors.New("bad transport: " + route.Adapter)
	}
	conn, err := transport.Dial(route.Address, route.Options)
	if err != nil {
		return nil, err
	}

	format := getopt("SYSLOG_FORMAT", "rfc5424")
	priority := getopt("SYSLOG_PRIORITY", "{{.Priority}}")
	hostname := getopt("SYSLOG_HOSTNAME", "{{.Container.Config.Hostname}}")
	pid := getopt("SYSLOG_PID", "{{.Container.State.Pid}}")
	tag := getopt("SYSLOG_TAG", "{{.ContainerName}}"+route.Options["append_tag"])
	structuredData := getopt("SYSLOG_STRUCTURED_DATA", "")
	if route.Options["structured_data"] != "" {
		structuredData = route.Options["structured_data"]
	}
	data := getopt("SYSLOG_DATA", "{{.Data}}")

	if structuredData == "" {
		structuredData = "-"
	} else {
		structuredData = fmt.Sprintf("[%s]", structuredData)
	}

	var tmplStr string
	switch format {
	case "rfc5424":
		tmplStr = fmt.Sprintf("<%s>1 {{.Timestamp}} %s %s %s - %s %s\n",
			priority, hostname, tag, pid, structuredData, data)
	case "rfc3164":
		tmplStr = fmt.Sprintf("<%s>{{.Timestamp}} %s %s[%s]: %s\n",
			priority, hostname, tag, pid, data)
	default:
		return nil, errors.New("unsupported syslog format: " + format)
	}
	tmpl, err := template.New("syslog").Parse(tmplStr)
	if err != nil {
		return nil, err
	}
	return &SyslogAdapter{
		route: route,
		trnsp: transport,
		conn:  conn,
		tmpl:  tmpl,
	}, nil
}

type SyslogAdapter struct {
	mu    sync.Mutex
	conn  net.Conn
	trnsp router.AdapterTransport
	route *router.Route
	tmpl  *template.Template
}

func (a *SyslogAdapter) maybeReconnect(writeErr error) bool {
	nErr, ok := writeErr.(*net.OpError)
	if !ok || (!nErr.Temporary() && nErr.Err != syscall.EPIPE) {
		return false
	}
	time.Sleep(500 * time.Millisecond)
	conn, err := a.trnsp.Dial(a.route.Address, a.route.Options)
	if err != nil {
		log.Println("syslog:", err)
		return false
	}

	a.mu.Lock()
	a.conn.Close()
	a.conn = conn
	a.mu.Unlock()
	return true
}

func (a *SyslogAdapter) Stream(logstream chan *router.Message) {
	for message := range logstream {
		m := &SyslogMessage{message}
		buf, err := m.Render(a.tmpl)
		if err != nil {
			log.Println("syslog:", err)
			return
		}
		_, err = a.conn.Write(buf)
		if err != nil {
			log.Println("syslog:", err)
			if reflect.TypeOf(a.conn).String() != "*net.UDPConn" {
				if ok := a.maybeReconnect(err); !ok {
					return
				}
			}
		}
	}
}

type SyslogMessage struct {
	*router.Message
}

func (m *SyslogMessage) Render(tmpl *template.Template) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := tmpl.Execute(buf, m)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *SyslogMessage) Priority() syslog.Priority {
	switch m.Message.Source {
	case "stdout":
		return syslog.LOG_USER | syslog.LOG_INFO
	case "stderr":
		return syslog.LOG_USER | syslog.LOG_ERR
	default:
		return syslog.LOG_DAEMON | syslog.LOG_INFO
	}
}

func (m *SyslogMessage) Hostname() string {
	return hostname
}

func (m *SyslogMessage) Timestamp() string {
	return m.Message.Time.Format(time.RFC3339)
}

func (m *SyslogMessage) ContainerName() string {
	return m.Message.Container.Name[1:]
}
