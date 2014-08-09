package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/syslog"
	"net"
	"net/url"
	"strings"
)

func streamer(route *Route, logstream chan *Log) {
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
			switch u, err := url.Parse(addr); {
			case err != nil:
				debug(err)
				route.backends.Remove(addr)
				continue
			case u.Scheme == "udp":
				if err := udpStreamer(logline, u.Host); err != nil {
					debug("Send to", u.Host, "by udp failed", err)
					continue
				}
			case u.Scheme == "tcp":
				if err := tcpStreamer(logline, u.Host); err != nil {
					debug("Send to", u.Host, "by tcp failed", err)
					continue
				}
			case u.Scheme == "syslog":
				if err := syslogStreamer(logline, u.Host); err != nil {
					debug("Sent to syslog failed", err)
					continue
				}
			}
			break
		}
	}
}

func syslogStreamer(logline *Log, addr string) error {
	tag := fmt.Sprintf("%s.%s", logline.Appname, logline.Tag)
	remote, err := syslog.Dial("udp", addr, syslog.LOG_USER|syslog.LOG_INFO, tag)
	if err != nil {
		return err
	}
	io.WriteString(remote, logline.Data)
	return nil
}

func tcpStreamer(logline *Log, addr string) error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		debug("Resolve tcp failed", err)
		return err
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		debug("Connect backend failed", err)
		return err
	}
	defer conn.Close()
	writeJSON(conn, logline)
	return nil
}

func udpStreamer(logline *Log, addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		debug("Resolve udp failed", err)
		return err
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		debug("Connect backend failed", err)
		return err
	}
	defer conn.Close()
	writeJSON(conn, logline)
	return nil
}

func writeJSON(w io.Writer, logline *Log) {
	encoder := json.NewEncoder(w)
	encoder.Encode(logline)
}
