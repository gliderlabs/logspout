package httpstream

import (
	"github.com/gliderlabs/logspout/router"
	"github.com/gorilla/mux"
	"net/http"
)

func init() {
	logs := mux.NewRouter()
	logsHandler := func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		source := new(router.Source)
		switch {
		case params["predicate"] == "id" && params["value"] != "":
			source.ID = params["value"]
			if len(source.ID) > 12 {
				source.ID = source.ID[:12]
			}
		case params["predicate"] == "name" && params["value"] != "":
			source.Name = params["value"]
		case params["predicate"] == "filter" && params["value"] != "":
			source.Filter = params["value"]
		}

		if source.ID != "" && router.Attacher.Get(source.ID) == nil {
			http.NotFound(w, req)
			return
		}

		logstream := make(chan *router.Log)
		defer close(logstream)

		var closer <-chan bool
		if req.Header.Get("Upgrade") == "websocket" {
			closerBi := make(chan bool)
			go router.WebsocketStreamer(w, req, logstream, closerBi)
			closer = closerBi
		} else {
			go router.HttpStreamer(w, req, logstream, source.All() || source.Filter != "")
			closer = w.(http.CloseNotifier).CloseNotify()
		}

		router.Attacher.Listen(source, logstream, closer)
	}
	logs.HandleFunc("/logs/{predicate:[a-zA-Z]+}:{value}", logsHandler).Methods("GET")
	logs.HandleFunc("/logs", logsHandler).Methods("GET")

	http.Handle("/logs", logs)
	http.Handle("/logs/", logs)
}
