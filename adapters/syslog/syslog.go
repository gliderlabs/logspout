package syslog

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"log/syslog"
	"net"
	"os"
	"text/template"
	"time"

	"github.com/gliderlabs/logspout/router"
)

func init() {
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
		return nil, errors.New("unable to find adapter: " + route.Adapter)
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

	var tmplStr string
	switch format {
	case "rfc5424":
		tmplStr = fmt.Sprintf("<%d>1 {{.Timestamp}} %s %s %d - [%s] %s\n",
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
		conn:  conn,
		tmpl:  tmpl,
	}, nil
}

type SyslogAdapter struct {
	conn  net.Conn
	route *router.Route
	tmpl  *template.Template
}

func (a *SyslogAdapter) Stream(logstream chan *router.Message) {
	for message := range logstream {
		buf := new(bytes.Buffer)
		err := a.tmpl.Execute(buf, &SyslogMessage{message, a})
		if err != nil {
			log.Println("syslog:", err)
			a.route.Close()
			return
		}
		_, err = a.conn.Write(buf.Bytes())
		if err != nil {
			log.Println("syslog:", err)
			a.route.Close()
			return
		}
	}
}

type SyslogMessage struct {
	*router.Message
	adapter *SyslogAdapter
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
	h, _ := os.Hostname()
	return h
}

func (m *SyslogMessage) LocalAddr() string {
	return m.adapter.conn.LocalAddr().String()
}

func (m *SyslogMessage) Timestamp() string {
	return m.Message.Time.Format(time.RFC3339)
}

func (m *SyslogMessage) ContainerName() string {
	return m.Message.Container.Name[1:]
}
