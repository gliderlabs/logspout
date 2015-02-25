package router

import (
	"encoding/json"
	"fmt"
	"io"
	"log/syslog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/go.net/websocket"
)

type IgnorantWriter struct {
	original io.Writer
}

func (w IgnorantWriter) Write(p []byte) (int, error) {
	n, _ := w.original.Write(p)
	return n, nil
}

type Colorizer map[string]int

// returns up to 14 color escape codes (then repeats) for each unique key
func (c Colorizer) Get(key string) string {
	i, exists := c[key]
	if !exists {
		c[key] = len(c)
		i = c[key]
	}
	bright := "1;"
	if i%14 > 6 {
		bright = ""
	}
	return "\x1b[" + bright + "3" + strconv.Itoa(7-(i%7)) + "m"
}

func syslogStreamer(target Target, types []string, logstream chan *Log) {
	typestr := "," + strings.Join(types, ",") + ","
	for logline := range logstream {
		if typestr != ",," && !strings.Contains(typestr, logline.Type) {
			continue
		}
		tag := logline.Name + target.AppendTag
		remote, err := syslog.Dial("udp", target.Addr, syslog.LOG_USER|syslog.LOG_INFO, tag)
		assert(err, "syslog")
		io.WriteString(remote, logline.Data)
	}
}

func udpStreamer(target Target, types []string, logstream chan *Log) {
	typestr := "," + strings.Join(types, ",") + ","
	addr, err := net.ResolveUDPAddr("udp", target.Addr)
	assert(err, "resolve udp failed")
	conn, err := net.DialUDP("udp", nil, addr)
	assert(err, "connect udp failed")
	encoder := json.NewEncoder(&IgnorantWriter{conn})
	defer conn.Close()
	for logline := range logstream {
		if typestr != ",," && !strings.Contains(typestr, logline.Type) {
			continue
		}
		encoder.Encode(logline)
	}
}

func rfc5424Streamer(target Target, types []string, logstream chan *Log) {
	typestr := "," + strings.Join(types, ",") + ","

	pri := syslog.LOG_USER | syslog.LOG_INFO
	hostname, _ := os.Hostname()

	c, err := net.Dial("udp", target.Addr)
	assert(err, "net dial rfc5424")

	if hostname == "" {
		hostname = c.LocalAddr().String()
	}

	for logline := range logstream {
		if typestr != ",," && !strings.Contains(typestr, logline.Type) {
			continue
		}
		tag := logline.Name + target.AppendTag
		nl := ""
		if !strings.HasSuffix(logline.Data, "\n") {
			nl = "\n"
		}

		timestamp := time.Now().Format(time.RFC3339)
		_, err := fmt.Fprintf(c, "<%d>1 %s %s %s %d - [%s] %s%s", pri, timestamp, hostname, tag, os.Getpid(), target.StructuredData, logline.Data, nl)
		assert(err, "rfc5424")
	}
}

func WebsocketStreamer(w http.ResponseWriter, req *http.Request, logstream chan *Log, closer chan bool) {
	websocket.Handler(func(conn *websocket.Conn) {
		for logline := range logstream {
			if req.URL.Query().Get("type") != "" && logline.Type != req.URL.Query().Get("type") {
				continue
			}
			_, err := conn.Write(append(marshal(logline), '\n'))
			if err != nil {
				closer <- true
				return
			}
		}
	}).ServeHTTP(w, req)
}

func HttpStreamer(w http.ResponseWriter, req *http.Request, logstream chan *Log, multi bool) {
	var colors Colorizer
	var usecolor, usejson bool
	nameWidth := 16
	if req.URL.Query().Get("colors") != "off" {
		colors = make(Colorizer)
		usecolor = true
	}
	if req.Header.Get("Accept") == "application/json" {
		w.Header().Add("Content-Type", "application/json")
		usejson = true
	} else {
		w.Header().Add("Content-Type", "text/plain")
	}
	for logline := range logstream {
		if req.URL.Query().Get("types") != "" && logline.Type != req.URL.Query().Get("types") {
			continue
		}
		if usejson {
			w.Write(append(marshal(logline), '\n'))
		} else {
			if multi {
				if len(logline.Name) > nameWidth {
					nameWidth = len(logline.Name)
				}
				if usecolor {
					w.Write([]byte(fmt.Sprintf(
						"%s%"+strconv.Itoa(nameWidth)+"s|%s\x1b[0m\n",
						colors.Get(logline.Name), logline.Name, logline.Data,
					)))
				} else {
					w.Write([]byte(fmt.Sprintf(
						"%"+strconv.Itoa(nameWidth)+"s|%s\n", logline.Name, logline.Data,
					)))
				}
			} else {
				w.Write(append([]byte(logline.Data), '\n'))
			}
		}
		w.(http.Flusher).Flush()
	}
}
