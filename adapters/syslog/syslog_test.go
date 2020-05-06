package syslog

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/gliderlabs/logspout/transports/tcp"
	_ "github.com/gliderlabs/logspout/transports/tls"
	_ "github.com/gliderlabs/logspout/transports/udp"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/gliderlabs/logspout/router"
)

const (
	connCloseIdx = 5
)

var (
	container = &docker.Container{
		ID:   "8dfafdbc3a40",
		Name: "\x00container",
		Config: &docker.Config{
			Hostname: "8dfafdbc3a40",
		},
	}
	hostHostnameFilename = "/etc/host_hostname"
	hostnameContent      = "hostname"
	badHostnameContent   = "hostname\r\n"
)

func TestSyslogOctetFraming(t *testing.T) {
	os.Setenv("SYSLOG_TCP_FRAMING", "octet-counted")
	defer os.Unsetenv("SYSLOG_TCP_FRAMING")

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
	defer adapter.(*Adapter).conn.Close()

	stream := make(chan *router.Message)
	go adapter.Stream(stream)

	count := 1
	messages := make(chan string, count)
	go sendLogstream(stream, messages, adapter, count)

	timeout := time.After(6 * time.Second)
	msgnum := 1
	select {
	case msg := <-done:
		sizeStr := ""
		_, err := fmt.Sscan(msg, &sizeStr)
		if err != nil {
			t.Fatal("unable to scan size from message: ", err)
		}

		size, err := strconv.ParseInt(sizeStr, 10, 32)
		if err != nil {
			t.Fatal("unable to scan size from message: ", err)
		}

		expectedOctetFrame := len(sizeStr) + 1 + int(size)
		if len(msg) != expectedOctetFrame {
			t.Errorf("expected octet frame to be %d. got %d instead for message %s", expectedOctetFrame, size, msg)
		}
		return
	case <-timeout:
		t.Fatal("timeout after", msgnum, "messages")
		return
	}
}

func TestSyslogRetryCount(t *testing.T) {
	const defaultRetryCount = uint(10)
	newRetryCount := uint(20)
	os.Setenv("RETRY_COUNT", strconv.Itoa(int(newRetryCount)))
	retryCount, _ := getRetryCount()
	if retryCount != newRetryCount {
		t.Errorf("expected %v got %v", newRetryCount, retryCount)
	}

	os.Unsetenv("RETRY_COUNT")
	retryCount, _ = getRetryCount()
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
				<-messages
				msgnum++
			}
			check(t, <-messages, msg)
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

func TestHostnameDoesNotHaveLineFeed(t *testing.T) {
	if err := ioutil.WriteFile(hostHostnameFilename, []byte(badHostnameContent), 0777); err != nil {
		t.Fatal(err)
	}
	testHostname := getHostname()
	if strings.Contains(testHostname, badHostnameContent) {
		t.Errorf("expected hostname to be %s. got %s in hostname %s", hostnameContent, badHostnameContent, testHostname)
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
		b, _ := msg.Render(adapter.(*Adapter).format, adapter.(*Adapter).tmpl)
		messages <- string(b)
		time.Sleep(10 * time.Millisecond)
	}
}

func check(t *testing.T, in string, out string) {
	if in != out {
		t.Errorf("expected: %s\ngot: %s\n", in, out)
	}
}
