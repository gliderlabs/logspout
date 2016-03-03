package routesapi

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gliderlabs/logspout/router"
	"github.com/gorilla/mux"
)

func init() {
	router.HttpHandlers.Register(RoutesAPI, "routes")
}

func RoutesAPI() http.Handler {
	routes := router.Routes
	r := mux.NewRouter()

	r.HandleFunc("/routes/{id}", func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		route, _ := routes.Get(params["id"])
		if route == nil {
			http.NotFound(w, req)
			return
		}
		w.Write(append(marshal(route), '\n'))
	}).Methods("GET")

	r.HandleFunc("/routes/{id}", func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		if ok := routes.Remove(params["id"]); !ok {
			http.NotFound(w, req)
		}
	}).Methods("DELETE")

	r.HandleFunc("/routes", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		rts, _ := routes.GetAll()
		w.Write(append(marshal(rts), '\n'))
		return
	}).Methods("GET")

	r.HandleFunc("/routes", func(w http.ResponseWriter, req *http.Request) {
		route := new(router.Route)
		if err := unmarshal(req.Body, route); err != nil {
			http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
		err := routes.Add(route)
		if err != nil {
			http.Error(w, "Bad route: "+err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(append(marshal(route), '\n'))
	}).Methods("POST")

	return r
}

func marshal(obj interface{}) []byte {
	bytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		log.Println("marshal:", err)
	}
	return bytes
}

func unmarshal(input io.Reader, obj interface{}) error {
	dec := json.NewDecoder(input)
	if err := dec.Decode(obj); err != nil {
		return err
	}
	return nil
}
