package main

import (
	"encoding/json"
	"log"
	"net"
	"strings"
)

func udpStreamer(route *Route, logstream chan *Log) {
	var types map[string]struct{}
	if route.Source != nil {
		types = make(map[string]struct{})
		for _, t := range route.Source.Types {
			types[t] = struct{}{}
		}
	}
	for logline := range logstream {
		if types != nil {
			if _, ok := types[logline.Type]; !ok {
				continue
			}
		}
		appinfo := strings.SplitN(logline.Name, "_", 2)
		logline.Appname = appinfo[0]
		logline.Port = appinfo[1]
		logline.Tag = route.Target.AppendTag

		addr, err := route.backends.Get(logline.Appname)
		debug(logline.Appname, addr)
		if err != nil {
			debug("Get backend failed", err)
			log.Println(logline.Appname, logline.Data)
			continue
		}

		udpAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			debug("Resolve udp failed", err)
			continue
		}

		var conn *net.UDPConn
		conn, err = net.DialUDP("udp", nil, udpAddr)
		if err != nil {
			debug("Connect backend failed", err)
			continue
		}
		defer conn.Close()
		encoder := json.NewEncoder(conn)
		encoder.Encode(logline)
	}
}
