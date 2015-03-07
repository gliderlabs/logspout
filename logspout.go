package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/fsouza/go-dockerclient"

	"github.com/gliderlabs/logspout/router"
)

var Version string

func debug(v ...interface{}) {
	if getopt("DEBUG", "") != "" {
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

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println(Version)
		os.Exit(0)
	}

	port := getopt("PORT", "8000")
	endpoint := getopt("DOCKER_HOST", "unix:///tmp/docker.sock")
	routespath := getopt("ROUTESPATH", "/mnt/routes")

	client, err := docker.NewClient(endpoint)
	assert(err, "docker")
	pump := router.NewLogsPump(client)
	routes := router.NewRouteManager(pump)

	var uris string
	if os.Getenv("ROUTE_URIS") != "" {
		uris = os.Getenv("ROUTE_URIS")
	}
	if len(os.Args) > 1 {
		uris = os.Args[1]
	}
	if uris != "" {
		for _, uri := range strings.Split(uris, ",") {
			err := routes.AddFromUri(uri)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Println("routing all to " + uri)
		}
	}

	if _, err := os.Stat(routespath); err == nil {
		log.Println("loading and persisting routes in " + routespath)
		assert(routes.Load(router.RouteFileStore(routespath)), "persistor")
	}

	for name, handler := range router.HttpHandlers.All() {
		h := handler(routes, pump)
		http.Handle("/"+name, h)
		http.Handle("/"+name+"/", h)
	}

	go pump.Pump()
	log.Printf("logspout %s serving http on :%s", Version, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
