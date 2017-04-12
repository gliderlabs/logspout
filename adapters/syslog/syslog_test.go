package syslog

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/gliderlabs/logspout/router"

	_ "github.com/gliderlabs/logspout/transports/tcp"
	_ "github.com/gliderlabs/logspout/transports/tls"
	_ "github.com/gliderlabs/logspout/transports/udp"
)

const (
	closeOnMsgIdx = 5
	maxMsgCount   = 10
	testPriority  = "{{.Priority}}"
	testTimestamp = "{{.Timestamp}}"
	testHostname  = "{{.Container.Config.Hostname}}"
	testTag       = "{{.ContainerName}}"
	testPid       = "{{.Container.State.Pid}}"
	testData      = "{{.Data}}"
)

var (
	container = &docker.Container{
		ID:   "8dfafdbc3a40",
		Name: "0michaelshobbs",
		Config: &docker.Config{
			Hostname: "8dfafdbc3a40",
		},
	}
	testTmplStr = fmt.Sprintf("<%s>%s %s %s[%s]: %s\n",
		testPriority, testTimestamp, testHostname, testTag, testPid, testData)
)

type localTCPServer struct {
	lnmu sync.RWMutex
	net.Listener
}

func (ls *localTCPServer) teardown() error {
	ls.lnmu.Lock()
	if ls.Listener != nil {
		ls.Listener.Close()
		ls.Listener = nil
	}
	ls.lnmu.Unlock()
	return nil
}

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
	os.Setenv("RETRY_COUNT", strconv.Itoa(int(1)))
	setRetryCount()
	defer func() {
		os.Unsetenv("RETRY_COUNT")
		setRetryCount()
	}()
	tmpl, err := template.New("syslog").Parse(testTmplStr)
	if err != nil {
		t.Fatal(err)
	}
	ls, err := newLocalTCPServer()
	if err != nil {
		t.Fatal(err)
	}
	defer ls.teardown()

	route := &router.Route{
		ID:      "0",
		Adapter: "syslog",
		Address: ls.Listener.Addr().String(),
	}
	transport, found := router.AdapterTransports.Lookup(route.AdapterTransport("tcp"))
	if !found {
		t.Errorf("bad transport: " + route.Adapter)
	}

	datac := make(chan []byte, maxMsgCount)
	errc := make(chan error, 1)
	go acceptAndCloseConn(ls, datac, errc)

	// Dial connection for adapter
	conn, err := net.Dial(ls.Listener.Addr().Network(), ls.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	adapter := &Adapter{
		route:     route,
		conn:      conn,
		tmpl:      tmpl,
		transport: transport,
	}

	logstream := make(chan *router.Message)
	done := make(chan bool)
	sentMsgs := [][]byte{}
	// Send msgs to logstream
	go sendLogstream(logstream, done, &sentMsgs, tmpl)

	// Stream logstream to conn
	go adapter.Stream(logstream)

	for err := range errc {
		t.Errorf("%v", err)
	}

	readMsgs := [][]byte{}
	for {
		select {
		case <-done:
			if maxMsgCount-1 != len(datac) {
				t.Errorf("expected %v got %v", maxMsgCount-1, len(datac))
			}
			for msg := range datac {
				readMsgs = append(readMsgs, msg)
			}
			sentMsgs = append(sentMsgs[:closeOnMsgIdx], sentMsgs[closeOnMsgIdx+1:]...)
			for i, v := range sentMsgs {
				sent := strings.Trim(fmt.Sprintf("%s", v), "\n")
				read := strings.Trim(fmt.Sprintf("%s", readMsgs[i]), "\x00\n")
				if sent != read {
					t.Errorf("expected %+q got %+q", sent, read)
				}
			}
		}
		return
	}
}

func newLocalTCPServer() (*localTCPServer, error) {
	ln, err := newLocalListener()
	if err != nil {
		return nil, err
	}
	return &localTCPServer{Listener: ln}, nil
}

func newLocalListener() (net.Listener, error) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	return ln, nil
}

func acceptAndCloseConn(ls *localTCPServer, datac chan []byte, errc chan error) {
	defer func() {
		close(datac)
		close(errc)
	}()
	c, err := ls.Accept()
	if err != nil {
		errc <- err
		return
	}
	count := 0
	for {
		switch count {
		case maxMsgCount - closeOnMsgIdx:
			c.Close()
			c, err = ls.Accept()
			if err != nil {
				errc <- err
				return
			}
			c.SetReadDeadline(time.Now().Add(5 * time.Second))
			readConn(c, datac)
			count++
		case maxMsgCount:
			return
		default:
			readConn(c, datac)
			count++
		}
	}
}

func readConn(c net.Conn, ch chan []byte) error {
	b := make([]byte, 256)
	_, err := c.Read(b)
	if err != nil {
		return err
	}
	ch <- b
	return nil
}

func sendLogstream(logstream chan *router.Message, done chan bool, msgs *[][]byte, tmpl *template.Template) {
	defer func() {
		close(logstream)
		close(done)
	}()
	var count int
	for {
		if count == maxMsgCount {
			done <- true
			return
		}
		msg := &router.Message{
			Container: container,
			Data:      "hellooo",
			Time:      time.Now(),
		}
		m := &Message{msg}
		buf, _ := m.Render(tmpl)
		*msgs = append(*msgs, buf)
		logstream <- msg
		count++
		time.Sleep(250 * time.Millisecond)
	}
}
