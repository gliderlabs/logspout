package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
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
	router.Attacher = router.NewAttachManager(client)
	router.Router = router.NewRouteManager(router.Attacher)

	if len(os.Args) > 1 {
		routes := strings.Split(os.Args[1], ",")
		for _, route := range routes {
			u, err := url.Parse(route)
			assert(err, "url")
			log.Println("routing all to " + route)

			r := router.Route{
				Target: router.Target{
					Type: u.Scheme,
					Addr: u.Host,
				},
			}
			if u.RawQuery != "" {
				v, err := url.ParseQuery(u.RawQuery)
				assert(err, "query")

				if v.Get("filter") != "" || v.Get("types") != "" {
					r.Source = &router.Source{
						Filter: v.Get("filter"),
						Types:  strings.Split(v.Get("types"), ","),
					}
				}

				r.Target.StructuredData = v.Get("structuredData")
				r.Target.AppendTag = v.Get("appendTag")
			}
			router.Router.Add(&r)
		}
	}

	if _, err := os.Stat(routespath); err == nil {
		log.Println("loading and persisting routes in " + routespath)
		assert(router.Router.Load(router.RouteFileStore(routespath)), "persistor")
	}

	log.Printf("logspout %s serving http on :%s", Version, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
