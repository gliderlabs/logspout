package httpstream

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"code.google.com/p/go.net/websocket"
	"github.com/gorilla/mux"

	"github.com/gliderlabs/logspout/router"
)

func init() {
	router.HttpHandlers.Register(LogStreamer, "logs")
}

func LogStreamer(routes *router.RouteManager, pump router.LogRouter) http.Handler {
	logs := mux.NewRouter()
	logsHandler := func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		route := new(router.Route)
		switch {
		case params["predicate"] == "id" && params["value"] != "":
			route.FilterID = params["value"]
			if len(route.ID) > 12 {
				route.FilterID = route.FilterID[:12]
			}
		case params["predicate"] == "name" && params["value"] != "":
			route.FilterName = params["value"]
		}

		if route.FilterID != "" && !pump.RoutingFrom(route.FilterID) {
			http.NotFound(w, req)
			return
		}

		logstream := make(chan *router.Message)
		defer close(logstream)

		var closer <-chan bool
		if req.Header.Get("Upgrade") == "websocket" {
			closerBi := make(chan bool)
			go websocketStreamer(w, req, logstream, closerBi)
			closer = closerBi
		} else {
			go httpStreamer(w, req, logstream, route.MultiContainer())
			closer = w.(http.CloseNotifier).CloseNotify()
		}
		route.OverrideCloser(closer)

		pump.Route(route, logstream)
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
