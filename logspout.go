package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/url"
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

func udpStreamer(target Target, types []string, logstream chan *Log) {
	typestr := "," + strings.Join(types, ",") + ","
	addr, err := net.ResolveUDPAddr("udp", target.Addr)
	assert(err, "resolve udp failed")
	conn, err := net.DialUDP("udp", nil, addr)
	assert(err, "connect udp failed")
	encoder := json.NewEncoder(conn)
	defer conn.Close()
	for logline := range logstream {
		if typestr != ",," && !strings.Contains(typestr, logline.Type) {
			continue
		}
		appinfo := strings.SplitN(logline.Name, "_", 2)
		logline.Appname = appinfo[0]
		logline.Port = appinfo[1]
		logline.Tag = target.AppendTag
		encoder.Encode(logline)
	}
}

func main() {
	flag.BoolVar(&debugMode, "DEBUG", false, "enable debug")
	endpoint := flag.String("docker", "unix:///var/run/docker.sock", "docker location")
	routes := flag.String("routes", "/var/lib/lenz", "routes path")
	forwarder := flag.String("forwarder", "udp://127.0.0.1:20000", "log forward dest")
	pidFile := flag.String("pidfile", "/var/run/lenz.pid", "pid file")
	flag.Parse()

	client, err := docker.NewClient(*endpoint)
	assert(err, "docker")
	attacher := NewAttachManager(client)
	router := NewRouteManager(attacher)

	u, err := url.Parse(*forwarder)
	assert(err, "url")
	log.Println("routing all to " + *forwarder)
	router.Add(&Route{Target: Target{Type: u.Scheme, Addr: u.Host}})

	if _, err := os.Stat(*routes); err == nil {
		log.Println("loading and persisting routes in " + *routes)
		assert(router.Load(RouteFileStore(*routes)), "persistor")
	}

	pid(*pidFile)
	var c = make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGHUP)
	for {
		log.Println("Catch", <-c)
		break
	}
}
