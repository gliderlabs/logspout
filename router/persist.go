package router

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type RouteFileStore string

func (fs RouteFileStore) Filename(id string) string {
	return string(fs) + "/" + id + ".json"
}

func (fs RouteFileStore) Get(id string) (*Route, error) {
	file, err := os.Open(fs.Filename(id))
	if err != nil {
		return nil, err
	}
	route := new(Route)
	if err = unmarshal(file, route); err != nil {
		return nil, err
	}
	return route, nil
}

func (fs RouteFileStore) GetAll() ([]*Route, error) {
	files, err := ioutil.ReadDir(string(fs))
	if err != nil {
		return nil, err
	}
	var routes []*Route
	for _, file := range files {
		fileparts := strings.Split(file.Name(), ".")
		if len(fileparts) > 1 && fileparts[1] == "json" {
			route, err := fs.Get(fileparts[0])
			if err == nil {
				routes = append(routes, route)
			}
		}
	}
	return routes, nil
}

func (fs RouteFileStore) Add(route *Route) error {
	return ioutil.WriteFile(fs.Filename(route.ID), marshal(route), 0644)
}

func (fs RouteFileStore) Remove(id string) bool {
	if _, err := os.Stat(fs.Filename(id)); err == nil {
		if err := os.Remove(fs.Filename(id)); err != nil {
			return true
		}
	}
	return false
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
