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

		for offset := 0; offset < route.backends.Len(); offset++ {
			addr, err := route.backends.Get(logline.Appname, offset)
			if err != nil {
				debug("Get backend failed", err)
				log.Println(logline.Appname, logline.Data)
				break
			}

			debug(logline.Appname, addr)
			udpAddr, err := net.ResolveUDPAddr("udp", addr)
			if err != nil {
				debug("Resolve udp failed", err)
				continue
			}

			conn, err := net.DialUDP("udp", nil, udpAddr)
			if err != nil {
				debug("Connect backend failed", err)
				continue
			}
			defer conn.Close()
			encoder := json.NewEncoder(conn)
			encoder.Encode(logline)
			break
		}
	}
}
