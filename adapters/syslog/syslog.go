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
	"text/template"
	"time"

	"github.com/gliderlabs/logspout/router"
)

var hostname string
var facility syslog.Priority

func init() {
	hostname, _ = os.Hostname()
	facility = get_facility()
	router.AdapterFactories.Register(NewSyslogAdapter, "syslog")
}

func getopt(name, dfault string) string {
	value := os.Getenv(name)
	if value == "" {
		value = dfault
	}
	return value
}

func get_facility() syslog.Priority {
	switch getopt("SYSLOG_FACILITY", "user") {
	case "kern":
		return syslog.LOG_KERN
	case "user":
		return syslog.LOG_USER
	case "mail":
		return syslog.LOG_MAIL
	case "daemon":
		return syslog.LOG_DAEMON
	case "auth":
		return syslog.LOG_AUTH
	case "syslog":
		return syslog.LOG_SYSLOG
	case "lpr":
		return syslog.LOG_LPR
	case "news":
		return syslog.LOG_NEWS
	case "uucp":
		return syslog.LOG_UUCP
	case "cron":
		return syslog.LOG_CRON
	case "authpriv":
		return syslog.LOG_AUTHPRIV
	case "ftp":
		return syslog.LOG_FTP
	case "local0":
		return syslog.LOG_LOCAL0
	case "local1":
		return syslog.LOG_LOCAL1
	case "local2":
		return syslog.LOG_LOCAL2
	case "local3":
		return syslog.LOG_LOCAL3
	case "local4":
		return syslog.LOG_LOCAL4
	case "local5":
		return syslog.LOG_LOCAL5
	case "local6":
		return syslog.LOG_LOCAL6
	case "local7":
		return syslog.LOG_LOCAL7
	}
	return syslog.LOG_USER
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

	var tmplStr string
	switch format {
	case "rfc5424":
		tmplStr = fmt.Sprintf("<%s>1 {{.Timestamp}} %s %s %s - [%s] %s\n",
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
				return
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
		return facility | syslog.LOG_INFO
	case "stderr":
		return facility | syslog.LOG_ERR
	default:
		return facility | syslog.LOG_INFO
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
