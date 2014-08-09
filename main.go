package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/CMGS/go-dockerclient"
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

func pid(path string) {
	if err := ioutil.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0755); err != nil {
		log.Println("Save app config failed", err)
	}
}

func main() {
	flag.BoolVar(&debugMode, "DEBUG", false, "enable debug")
	endpoint := flag.String("docker", "unix:///var/run/docker.sock", "docker location")
	routes := flag.String("routes", "/var/lib/lenz", "routes path")
	forwards := flag.String("forwards", "", "log forward location, separate by comma")
	pidFile := flag.String("pidfile", "/var/run/lenz.pid", "pid file")
	flag.Parse()

	client, err := docker.NewClient(*endpoint)
	assert(err, "docker")
	attacher := NewAttachManager(client)
	router := NewRouteManager(attacher)
	routefs := RouteFileStore(*routes)

	if *forwards != "" {
		log.Println("routing all to " + *forwards)
		target := Target{Addrs: strings.Split(*forwards, ",")}
		route := Route{ID: "lenz_default", Target: &target}
		route.loadBackends()
		router.Add(&route)
	}

	if _, err := os.Stat(*routes); err == nil {
		log.Println("loading and persisting routes in " + *routes)
		assert(router.Load(routefs), "persistor")
	}

	pid(*pidFile)
	var c = make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGHUP)
	for {
		s := <-c
		log.Println("Catch", s)
		switch s {
		case syscall.SIGHUP:
			assert(router.Reload(), "persistor")
		default:
			os.Exit(0)
		}
	}
}
