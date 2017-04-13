package syslog

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/gliderlabs/logspout/router"
	"github.com/gliderlabs/logspout/testutil"

	_ "github.com/gliderlabs/logspout/transports/tcp"
	_ "github.com/gliderlabs/logspout/transports/tls"
	_ "github.com/gliderlabs/logspout/transports/udp"
)

const (
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
		Name: "\x00michaelshobbs",
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
	ls, err := testutil.NewLocalTCPServer()
	if err != nil {
		t.Fatal(err)
	}
	defer ls.Teardown()

	route := &router.Route{
		ID:      "0",
		Adapter: "syslog",
		Address: ls.Listener.Addr().String(),
	}

	datac := make(chan []byte, testutil.MaxMsgCount)
	errc := make(chan error, 1)
	go testutil.AcceptAndCloseConn(ls, datac, errc)

	adapter := &Adapter{
		route:     route,
		conn:      testutil.Dial(ls),
		tmpl:      tmpl,
		transport: testutil.MockTransport{Listener: ls},
	}

	logstream := make(chan *router.Message)
	done := make(chan bool)
	sentMsgs := [][]byte{}
	// Send msgs to logstream
	go sendLogstream(logstream, done, &sentMsgs, tmpl)

	// Stream logstream to conn
	go adapter.Stream(logstream)

	// Check for errs from goroutines
	for err := range errc {
		t.Errorf("%v", err)
	}

	readMsgs := [][]byte{}
	for {
		select {
		case <-done:
			if testutil.MaxMsgCount-1 != len(datac) {
				t.Errorf("expected %v got %v", testutil.MaxMsgCount-1, len(datac))
			}
			for msg := range datac {
				readMsgs = append(readMsgs, msg)
			}
			sentMsgs = append(sentMsgs[:testutil.CloseOnMsgIdx], sentMsgs[testutil.CloseOnMsgIdx+1:]...)
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

func sendLogstream(logstream chan *router.Message, done chan bool, msgs *[][]byte, tmpl *template.Template) {
	defer func() {
		close(logstream)
		close(done)
	}()
	var count int
	for {
		if count == testutil.MaxMsgCount {
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
		time.Sleep(1000 * time.Millisecond)
	}
}
