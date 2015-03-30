package json

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/gliderlabs/logspout/adapters/syslog"
	"github.com/gliderlabs/logspout/router"
)

var configDefaults = map[string]string{
	"JSON_FIELDS":          "time:uint,message,docker.hostname,docker.image",
	"JSON_TIME":            "{{.Time.Unix}}",
	"JSON_MESSAGE":         "{{.Data}}",
	"JSON_SOURCE":          "{{.Source}}",
	"JSON_DOCKER_HOSTNAME": "{{.Container.Config.Hostname}}",
	"JSON_DOCKER_IMAGE":    "{{.Container.Config.Image}}",
	"JSON_DOCKER_ID":       "{{.Container.ID}}",
	"JSON_DOCKER_NAME":     "{{.ContainerName}}",
}

func init() {
	router.AdapterFactories.Register(NewJSONAdapter, "json")
}

func getopt(name string) string {
	value := os.Getenv(name)
	if value == "" {
		value = configDefaults[name]
	}
	return value
}

type JSONAdapter struct {
	conn  net.Conn
	route *router.Route
	tmpl  *template.Template
	types map[string]string
}

func NewJSONAdapter(route *router.Route) (router.LogAdapter, error) {
	transport, found := router.AdapterTransports.Lookup(route.AdapterTransport("udp"))
	if !found {
		return nil, errors.New("unable to find adapter: " + route.Adapter)
	}
	conn, err := transport.Dial(route.Address, route.Options)
	if err != nil {
		return nil, err
	}

	fields := strings.Split(getopt("JSON_FIELDS"), ",")
	types := make(map[string]string)
	var values []string
	for _, field := range fields {
		parts := strings.Split(field, ":")
		if len(parts) > 1 {
			types[parts[0]] = parts[1]
		}
		config := "JSON_" + strings.ToUpper(strings.Replace(parts[0], ".", "_", -1))
		values = append(values, parts[0]+":"+getopt(config))
	}
	tmplStr := strings.Join(values, "\x00")
	tmpl, err := template.New("prejson").Parse(tmplStr)
	if err != nil {
		return nil, err
	}

	return &JSONAdapter{
		route: route,
		conn:  conn,
		tmpl:  tmpl,
		types: types,
	}, nil
}

func (a *JSONAdapter) Stream(logstream chan *router.Message) {
	defer a.route.Close()
	for message := range logstream {
		m := syslog.NewSyslogMessage(message, a.conn)
		buf, err := m.Render(a.tmpl)
		if err != nil {
			log.Println("json:", err)
			return
		}
		data, err := json.Marshal(buildMap(buf.String(), a.types))
		if err != nil {
			log.Println("json:", err)
			return
		}
		_, err = a.conn.Write(data)
		if err != nil {
			log.Println("json:", err)
			return
		}
	}
}

func buildMap(input string, types map[string]string) map[string]interface{} {
	m := make(map[string]interface{})
	fields := strings.Split(input, "\x00")
	for _, field := range fields {
		kvp := strings.SplitN(field, ":", 2)
		keys := strings.Split(kvp[0], ".")
		mm := m
		if len(keys) > 1 {
			for _, key := range keys[:len(keys)-1] {
				if mm[key] == nil {
					mm[key] = make(map[string]interface{})
				}
				mm = mm[key].(map[string]interface{})
			}
		}
		var value interface{}
		var err error
		switch types[field] {
		case "uint":
			value, err = strconv.ParseUint(kvp[1], 10, 64)
		case "int":
			value, err = strconv.ParseInt(kvp[1], 10, 64)
		case "float":
			value, err = strconv.ParseFloat(kvp[1], 64)
		case "bool":
			value, err = strconv.ParseBool(kvp[1])
		case "":
			value = kvp[1]
		}
		if err != nil {
			value = nil
		}
		mm[keys[len(keys)-1]] = value
	}
	return m
}
