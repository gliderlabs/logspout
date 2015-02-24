package routesapi

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gliderlabs/logspout/router"
	"github.com/gorilla/mux"
)

func marshal(obj interface{}) []byte {
	bytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		log.Println("marshal:", err)
	}
	return bytes
}

func unmarshal(input io.ReadCloser, obj interface{}) error {
	body, err := ioutil.ReadAll(input)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, obj)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	routes := mux.NewRouter()
	routes.HandleFunc("/routes/{id}", func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		route, _ := router.Router.Get(params["id"])
		if route == nil {
			http.NotFound(w, req)
			return
		}
		w.Write(append(marshal(route), '\n'))
	}).Methods("GET")

	routes.HandleFunc("/routes/{id}", func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		if ok := router.Router.Remove(params["id"]); !ok {
			http.NotFound(w, req)
		}
	}).Methods("DELETE")

	routes.HandleFunc("/routes", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		routes, _ := router.Router.GetAll()
		w.Write(append(marshal(routes), '\n'))
		return
	}).Methods("GET")

	routes.HandleFunc("/routes", func(w http.ResponseWriter, req *http.Request) {
		route := new(router.Route)
		if err := unmarshal(req.Body, route); err != nil {
			http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
			return
		}

		// TODO: validate?
		router.Router.Add(route)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(append(marshal(route), '\n'))
	}).Methods("POST")

	http.Handle("/routes", routes)
	http.Handle("/routes/", routes)
}
