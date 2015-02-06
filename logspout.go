package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/syslog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/go.net/websocket"
	"github.com/fsouza/go-dockerclient"
	"github.com/go-martini/martini"
	"github.com/streadway/amqp"
)

var debugMode bool

func debug(v ...interface{}) {
	if debugMode {
		log.Println(v...)
	}
}

func assert(err error, context string) {
	if err != nil {
		log.Fatal(context+": ", err)
	}
}

func getopt(name, dfault string) string {
	value := os.Getenv(name)
	if value == "" {
		value = dfault
	}
	return value
}

type Colorizer map[string]int

// returns up to 14 color escape codes (then repeats) for each unique key
func (c Colorizer) Get(key string) string {
	i, exists := c[key]
	if !exists {
		c[key] = len(c)
		i = c[key]
	}
	bright := "1;"
	if i%14 > 6 {
		bright = ""
	}
	return "\x1b[" + bright + "3" + strconv.Itoa(7-(i%7)) + "m"
}


func rabbitmqStreamer(target Target, types []string, logstream chan *Log) {
	typestr := "," + strings.Join(types, ",") + ","

    // Connects opens an AMQP connection from the credentials in the URL.
    conn, err := amqp.Dial([]string{target.Addr})
    if err != nil {
       log.Fatalf("connection.open: %s", err)
    }

    // This waits for a server acknowledgment which means the sockets will have
    // flushed all outbound publishings prior to returning.  It's important to
    // block on Close to not lose any publishings.
    defer conn.Close()

    c, err := conn.Channel()
    if err != nil {
        log.Fatalf("channel.open: %s", err)
    }

    for logline := range logstream {
		if typestr != ",," && !strings.Contains(typestr, logline.Type) {
			continue
		}

// We declare our topology on both the publisher and consumer to ensure they
// are the same.  This is part of AMQP being a programmable messaging model.
//
// See the Channel.Consume example for the complimentary declare.
// UNCOMMENT ME WHEN THIS STUFF IS READY
//err = c.ExchangeDeclare("logstash_exchange", "topic", true, false, false, false, nil)
//if err != nil {
//    log.Fatalf("exchange.declare: %v", err)
//}

    // Prepare this message to be persistent.  Your publishing requirements may
    // be different.
    msg := amqp.Publishing{
        Headers: amqp.Table{},
        ContentType: "text/plain",
        ContentEncoding: "UTF-8",
    //    DeliveryMode: amqp.Transient,
        DeliveryMode: amqp.Persistent,
        Priority: 0,
        Timestamp:    time.Now(),
        Body:         []byte(logline.Data),
    }

    // This is not a mandatory delivery, so it will be dropped if there are no
    // queues bound to the logstash exchange.
    err = c.Publish("logstash_exchange", "info", false, false, msg)
    if err != nil {
        // Since publish is asynchronous this can happen if the network connection
        // is reset or if the server has run out of resources.
        log.Fatalf("basic.publish: %v", err)
    }
}

func syslogStreamer(target Target, types []string, logstream chan *Log) {
	typestr := "," + strings.Join(types, ",") + ","
	for logline := range logstream {
		if typestr != ",," && !strings.Contains(typestr, logline.Type) {
			continue
		}
		tag := logline.Name + target.AppendTag
		remote, err := syslog.Dial("udp", target.Addr, syslog.LOG_USER|syslog.LOG_INFO, tag)
		assert(err, "syslog")
		io.WriteString(remote, logline.Data)
	}
}

func udpStreamer(target Target, types []string, logstream chan *Log) {
	typestr := "," + strings.Join(types, ",") + ","
	addr, err := net.ResolveUDPAddr("udp", target.Addr)
	assert(err, "resolve udp failed")
	conn, err := net.DialUDP("udp", nil, addr)
	assert(err, "connect udp failed")
	encoder := json.NewEncoder(&IgnorantWriter{conn})
	defer conn.Close()
	for logline := range logstream {
		if typestr != ",," && !strings.Contains(typestr, logline.Type) {
			continue
		}
		encoder.Encode(logline)
	}
}

func rfc5424Streamer(target Target, types []string, logstream chan *Log) {
	typestr := "," + strings.Join(types, ",") + ","

	pri := syslog.LOG_USER | syslog.LOG_INFO
	hostname, _ := os.Hostname()

	c, err := net.Dial("udp", target.Addr)
	assert(err, "net dial rfc5424")

	if hostname == "" {
		hostname = c.LocalAddr().String()
	}

	for logline := range logstream {
		if typestr != ",," && !strings.Contains(typestr, logline.Type) {
			continue
		}
		tag := logline.Name + target.AppendTag
		nl := ""
		if !strings.HasSuffix(logline.Data, "\n") {
			nl = "\n"
		}

		timestamp := time.Now().Format(time.RFC3339)
		_, err := fmt.Fprintf(c, "<%d>1 %s %s %s %d - [%s] %s%s", pri, timestamp, hostname, tag, os.Getpid(), target.StructuredData, logline.Data, nl)
		assert(err, "rfc5424")
	}
}

func websocketStreamer(w http.ResponseWriter, req *http.Request, logstream chan *Log, closer chan bool) {
	websocket.Handler(func(conn *websocket.Conn) {
		for logline := range logstream {
			if req.URL.Query().Get("type") != "" && logline.Type != req.URL.Query().Get("type") {
				continue
			}
			_, err := conn.Write(append(marshal(logline), '\n'))
			if err != nil {
				closer <- true
				return
			}
		}
	}).ServeHTTP(w, req)
}

func httpStreamer(w http.ResponseWriter, req *http.Request, logstream chan *Log, multi bool) {
	var colors Colorizer
	var usecolor, usejson bool
	nameWidth := 16
	if req.URL.Query().Get("colors") != "off" {
		colors = make(Colorizer)
		usecolor = true
	}
	if req.Header.Get("Accept") == "application/json" {
		w.Header().Add("Content-Type", "application/json")
		usejson = true
	} else {
		w.Header().Add("Content-Type", "text/plain")
	}
	for logline := range logstream {
		if req.URL.Query().Get("types") != "" && logline.Type != req.URL.Query().Get("types") {
			continue
		}
		if usejson {
			w.Write(append(marshal(logline), '\n'))
		} else {
			if multi {
				if len(logline.Name) > nameWidth {
					nameWidth = len(logline.Name)
				}
				if usecolor {
					w.Write([]byte(fmt.Sprintf(
						"%s%"+strconv.Itoa(nameWidth)+"s|%s\x1b[0m\n",
						colors.Get(logline.Name), logline.Name, logline.Data,
					)))
				} else {
					w.Write([]byte(fmt.Sprintf(
						"%"+strconv.Itoa(nameWidth)+"s|%s\n", logline.Name, logline.Data,
					)))
				}
			} else {
				w.Write(append([]byte(logline.Data), '\n'))
			}
		}
		w.(http.Flusher).Flush()
	}
}

func main() {
	debugMode = getopt("DEBUG", "") != ""
	port := getopt("PORT", "8000")
	endpoint := getopt("DOCKER_HOST", "unix:///var/run/docker.sock")
	routespath := getopt("ROUTESPATH", "/var/lib/logspout")

	client, err := docker.NewClient(endpoint)
	assert(err, "docker")
	attacher := NewAttachManager(client)
	router := NewRouteManager(attacher)

	if len(os.Args) > 1 {
		u, err := url.Parse(os.Args[1])
		assert(err, "url")
		log.Println("routing all to " + os.Args[1])

		r := Route{
			Target: Target{
				Type: u.Scheme,
				Addr: u.Host,
			},
		}
		if u.RawQuery != "" {
			v, err := url.ParseQuery(u.RawQuery)
			assert(err, "query")

			if v.Get("filter") != "" || v.Get("types") != "" {
				r.Source = &Source{
					Filter: v.Get("filter"),
					Types:  strings.Split(v.Get("types"), ","),
				}
			}

			r.Target.StructuredData = v.Get("structuredData")
			r.Target.AppendTag = v.Get("appendTag")
		}

		router.Add(&r)
	}

	if _, err := os.Stat(routespath); err == nil {
		log.Println("loading and persisting routes in " + routespath)
		assert(router.Load(RouteFileStore(routespath)), "persistor")
	}

	m := martini.Classic()

	m.Get("/logs(?:/(?P<predicate>[a-zA-Z]+):(?P<value>.+))?", func(w http.ResponseWriter, req *http.Request, params martini.Params) {
		source := new(Source)
		switch {
		case params["predicate"] == "id" && params["value"] != "":
			source.ID = params["value"][:12]
		case params["predicate"] == "name" && params["value"] != "":
			source.Name = params["value"]
		case params["predicate"] == "filter" && params["value"] != "":
			source.Filter = params["value"]
		}

		if source.ID != "" && attacher.Get(source.ID) == nil {
			http.NotFound(w, req)
			return
		}

		logstream := make(chan *Log)
		defer close(logstream)

		var closer <-chan bool
		if req.Header.Get("Upgrade") == "websocket" {
			closerBi := make(chan bool)
			go websocketStreamer(w, req, logstream, closerBi)
			closer = closerBi
		} else {
			go httpStreamer(w, req, logstream, source.All() || source.Filter != "")
			closer = w.(http.CloseNotifier).CloseNotify()
		}

		attacher.Listen(source, logstream, closer)
	})

	m.Get("/routes", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		routes, _ := router.GetAll()
		w.Write(append(marshal(routes), '\n'))
	})

	m.Post("/routes", func(w http.ResponseWriter, req *http.Request) (int, string) {
		route := new(Route)
		if err := unmarshal(req.Body, route); err != nil {
			return http.StatusBadRequest, "Bad request: " + err.Error()
		}

		// TODO: validate?
		router.Add(route)

		w.Header().Add("Content-Type", "application/json")
		return http.StatusCreated, string(append(marshal(route), '\n'))
	})

	m.Get("/routes/:id", func(w http.ResponseWriter, req *http.Request, params martini.Params) {
		route, _ := router.Get(params["id"])
		if route == nil {
			http.NotFound(w, req)
			return
		}
		w.Write(append(marshal(route), '\n'))
	})

	m.Delete("/routes/:id", func(w http.ResponseWriter, req *http.Request, params martini.Params) {
		if ok := router.Remove(params["id"]); !ok {
			http.NotFound(w, req)
		}
	})

	log.Println("logspout serving http on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, m))
}
