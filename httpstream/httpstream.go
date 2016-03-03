package httpstream

import (
	"encoding/json"
	"fmt"
	"github.com/gliderlabs/logspout/router"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
	"log"
	"net/http"
	"os"
	"strconv"
)

func init() {
	router.HttpHandlers.Register(LogStreamer, "logs")
}

func debug(v ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		log.Println(v...)
	}
}

func LogStreamer() http.Handler {
	logs := mux.NewRouter()
	logsHandler := func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		route := new(router.Route)

		if params["value"] != "" {
			switch params["predicate"] {
			case "id":
				route.FilterID = params["value"]
				if len(route.ID) > 12 {
					route.FilterID = route.FilterID[:12]
				}
			case "name":
				route.FilterName = params["value"]
			}
		}

		if route.FilterID != "" && !router.Routes.RoutingFrom(route.FilterID) {
			http.NotFound(w, req)
			return
		}

		defer debug("http: logs streamer disconnected")
		logstream := make(chan *router.Message)
		defer close(logstream)

		var closer <-chan bool
		if req.Header.Get("Upgrade") == "websocket" {
			debug("http: logs streamer connected [websocket]")
			closerBi := make(chan bool)
			defer websocketStreamer(w, req, logstream, closerBi)
			closer = closerBi
		} else {
			debug("http: logs streamer connected [http]")
			defer httpStreamer(w, req, logstream, route.MultiContainer())
			closer = w.(http.CloseNotifier).CloseNotify()
		}
		route.OverrideCloser(closer)

		router.Routes.Route(route, logstream)
	}
	logs.HandleFunc("/logs/{predicate:[a-zA-Z]+}:{value}", logsHandler).Methods("GET")
	logs.HandleFunc("/logs", logsHandler).Methods("GET")
	return logs
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

func marshal(obj interface{}) []byte {
	bytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		log.Println("marshal:", err)
	}
	return bytes
}

func normalName(name string) string {
	return name[1:]
}

func websocketStreamer(w http.ResponseWriter, req *http.Request, logstream chan *router.Message, closer chan bool) {
	websocket.Handler(func(conn *websocket.Conn) {
		for logline := range logstream {
			if req.URL.Query().Get("source") != "" && logline.Source != req.URL.Query().Get("source") {
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

func httpStreamer(w http.ResponseWriter, req *http.Request, logstream chan *router.Message, multi bool) {
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
		if req.URL.Query().Get("sources") != "" && logline.Source != req.URL.Query().Get("sources") {
			continue
		}
		if usejson {
			w.Write(append(marshal(logline), '\n'))
		} else {
			if multi {
				name := normalName(logline.Container.Name)
				if len(name) > nameWidth {
					nameWidth = len(name)
				}
				if usecolor {
					w.Write([]byte(fmt.Sprintf(
						"%s%"+strconv.Itoa(nameWidth)+"s|%s\x1b[0m\n",
						colors.Get(name), name, logline.Data,
					)))
				} else {
					w.Write([]byte(fmt.Sprintf(
						"%"+strconv.Itoa(nameWidth)+"s|%s\n", name, logline.Data,
					)))
				}
			} else {
				w.Write(append([]byte(logline.Data), '\n'))
			}
		}
		w.(http.Flusher).Flush()
	}
}
