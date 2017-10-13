package syslog

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"testing"
	"text/template"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/deliveroo/logspout/router"

	_ "github.com/deliveroo/logspout/transports/tcp"
	_ "github.com/deliveroo/logspout/transports/tls"
	_ "github.com/deliveroo/logspout/transports/udp"
)

const (
	testPriority  = "{{.Priority}}"
	testTimestamp = "{{.Timestamp}}"
	testHostname  = "{{.Container.Config.Hostname}}"
	testTag       = "{{.ContainerName}}"
	testPid       = "{{.Container.State.Pid}}"
	testData      = "{{.Data}}"
	connCloseIdx  = 5
)

var (
	container = &docker.Container{
		ID:   "8dfafdbc3a40",
		Name: "\x00container",
		Config: &docker.Config{
			Hostname: "8dfafdbc3a40",
		},
	}
	testTmplStr = fmt.Sprintf("<%s>%s %s %s[%s]: %s\n",
		testPriority, testTimestamp, testHostname, testTag, testPid, testData)
)

func TestSyslogRetryCount(t *testing.T) {
	newRetryCount := uint(20)
	os.Setenv("RETRY_COUNT", strconv.Itoa(int(newRetryCount)))
	setRetryCount()
	if retryCount != newRetryCount {
		t.Errorf("expected %v got %v", newRetryCount, retryCount)
	}

	os.Unsetenv("RETRY_COUNT")
	setRetryCount()
	if retryCount != defaultRetryCount {
		t.Errorf("expected %v got %v", defaultRetryCount, retryCount)
	}
}

func TestSyslogReconnectOnClose(t *testing.T) {
	done := make(chan string)
	addr, sock, srvWG := startServer("tcp", "", done)
	defer srvWG.Wait()
	defer os.Remove(addr)
	defer sock.Close()
	route := &router.Route{Adapter: "syslog+tcp", Address: addr}
	adapter, err := NewSyslogAdapter(route)
	if err != nil {
		t.Fatal(err)
	}

	stream := make(chan *router.Message)
	go adapter.Stream(stream)

	count := 100
	messages := make(chan string, count)
	go sendLogstream(stream, messages, adapter, count)

	timeout := time.After(6 * time.Second)
	msgnum := 1
	for {
		select {
		case msg := <-done:
			// Don't check a message that we know was dropped
			if msgnum%connCloseIdx == 0 {
				_ = <-messages
				msgnum++
			}
			check(t, adapter.(*Adapter).tmpl, <-messages, msg)
			msgnum++
		case <-timeout:
			adapter.(*Adapter).conn.Close()
			t.Fatal("timeout after", msgnum, "messages")
			return
		default:
			if msgnum == count {
				adapter.(*Adapter).conn.Close()
				return
			}
		}
	}
}

func startServer(n, la string, done chan<- string) (addr string, sock io.Closer, wg *sync.WaitGroup) {
	if n == "udp" || n == "tcp" {
		la = "127.0.0.1:0"
	}
	wg = new(sync.WaitGroup)

	l, err := net.Listen(n, la)
	if err != nil {
		log.Fatalf("startServer failed: %v", err)
	}
	addr = l.Addr().String()
	sock = l
	wg.Add(1)
	go func() {
		defer wg.Done()
		runStreamSyslog(l, done, wg)
	}()

	return
}

func runStreamSyslog(l net.Listener, done chan<- string, wg *sync.WaitGroup) {
	for {
		c, err := l.Accept()
		if err != nil {
			return
		}
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			c.SetReadDeadline(time.Now().Add(5 * time.Second))
			b := bufio.NewReader(c)
			var i = 1
			for {
				i++
				s, err := b.ReadString('\n')
				if err != nil {
					break
				}
				done <- s
				if i%connCloseIdx == 0 {
					break
				}
			}
			c.Close()
		}(c)
	}
}

func sendLogstream(stream chan *router.Message, messages chan string, adapter router.LogAdapter, count int) {
	for i := 1; i <= count; i++ {
		msg := &Message{
			Message: &router.Message{
				Container: container,
				Data:      "test " + strconv.Itoa(i),
				Time:      time.Now(),
				Source:    "stdout",
			},
		}
		stream <- msg.Message
		b, _ := msg.Render(adapter.(*Adapter).tmpl)
		messages <- string(b)
		time.Sleep(10 * time.Millisecond)
	}
}

func check(t *testing.T, tmpl *template.Template, in string, out string) {
	if in != out {
		t.Errorf("expected: %s\ngot: %s\n", in, out)
	}
}
