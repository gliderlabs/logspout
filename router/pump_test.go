package router

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
)

type FakeRoundTripper struct {
	message  interface{}
	status   int
	header   map[string]string
	requests []*http.Request
}

func (rt *FakeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	b, err := json.Marshal(rt.message)
	if err != nil {
		log.Fatal(err.Error())
	}

	body := bytes.NewReader(b)
	rt.requests = append(rt.requests, r)
	res := &http.Response{
		StatusCode: rt.status,
		Body:       ioutil.NopCloser(body),
		Header:     make(http.Header),
	}
	for k, v := range rt.header {
		res.Header.Set(k, v)
	}
	return res, nil
}

func (rt *FakeRoundTripper) Reset() {
	rt.requests = nil
}

func newTestClient(rt *FakeRoundTripper) docker.Client {
	endpoint := "http://localhost:4243"
	client, _ := docker.NewClient(endpoint)
	client.HTTPClient = &http.Client{Transport: rt}
	client.Dialer = &net.Dialer{}
	client.SkipServerVersionCheck = true
	return *client
}

func TestPumpIgnoreContainer(t *testing.T) {
	os.Setenv("EXCLUDE_LABEL", "exclude")
	defer os.Unsetenv("EXCLUDE_LABEL")
	containers := []struct {
		in  *docker.Config
		out bool
	}{
		{&docker.Config{Env: []string{"foo", "bar"}}, false},
		{&docker.Config{Env: []string{"LOGSPOUT=ignore"}}, true},
		{&docker.Config{Env: []string{"LOGSPOUT=IGNORE"}}, true},
		{&docker.Config{Env: []string{"LOGSPOUT=foo"}}, false},
		{&docker.Config{Labels: map[string]string{"exclude": "true"}}, true},
		{&docker.Config{Labels: map[string]string{"exclude": "false"}}, false},
	}

	for _, conf := range containers {
		if actual := ignoreContainer(&docker.Container{Config: conf.in}); actual != conf.out {
			t.Errorf("expected %v got %v", conf.out, actual)
		}
	}
}

func TestPumpIgnoreContainerCustomLabels(t *testing.T) {
	os.Setenv("EXCLUDE_LABEL", "k8s-app:canal")
	defer os.Unsetenv("EXCLUDE_LABEL")
	containers := []struct {
		in  *docker.Config
		out bool
	}{
		{&docker.Config{Labels: map[string]string{"k8s-app": "canal"}}, true},
		{&docker.Config{Labels: map[string]string{"app": "demo-app"}}, false},
	}

	for _, conf := range containers {
		if actual := ignoreContainer(&docker.Container{Config: conf.in}); actual != conf.out {
			t.Errorf("expected %v got %v", conf.out, actual)
		}
	}
}

func TestPumpIgnoreContainerAllowTTYDefault(t *testing.T) {
	containers := []struct {
		in  *docker.Config
		out bool
	}{
		{&docker.Config{Tty: true}, true},
		{&docker.Config{Tty: false}, false},
	}

	for _, conf := range containers {
		if actual := ignoreContainerTTY(&docker.Container{Config: conf.in}); actual != conf.out {
			t.Errorf("expected %v got %v", conf.out, actual)
		}
	}
}

func TestPumpIgnoreContainerAllowTTYTrue(t *testing.T) {
	os.Setenv("ALLOW_TTY", "true")
	defer os.Unsetenv("ALLOW_TTY")

	setAllowTTY()
	containers := []struct {
		in  *docker.Config
		out bool
	}{
		{&docker.Config{Tty: true}, false},
		{&docker.Config{Tty: false}, false},
	}
	for _, conf := range containers {
		if actual := ignoreContainerTTY(&docker.Container{Config: conf.in}); actual != conf.out {
			t.Errorf("expected %v got %v", conf.out, actual)
		}
	}
}

func TestPumpLogsPumpName(t *testing.T) {
	p := &LogsPump{}
	if name := p.Name(); name != "pump" {
		t.Error("name should be 'pump' got:", name)
	}
}

func TestPumpContainerRename(t *testing.T) {
	container := &docker.Container{
		ID:   "8dfafdbc3a40",
		Name: "bar",
	}
	client := newTestClient(&FakeRoundTripper{message: container, status: http.StatusOK})
	p := &LogsPump{
		client: &client,
		pumps:  make(map[string]*containerPump),
		routes: make(map[chan *update]struct{}),
	}
	config := &docker.Config{
		Tty: false,
	}
	container = &docker.Container{
		ID:     "8dfafdbc3a40",
		Name:   "foo",
		Config: config,
	}
	p.pumps["8dfafdbc3a40"] = newContainerPump(container, os.Stdout, os.Stderr)
	if name := p.pumps["8dfafdbc3a40"].container.Name; name != "foo" {
		t.Errorf("containerPump should have name: 'foo' got name: '%s'", name)
	}
	p.rename(&docker.APIEvents{ID: "8dfafdbc3a40"})
	if name := p.pumps["8dfafdbc3a40"].container.Name; name != "bar" {
		t.Errorf("containerPump should have name: 'bar' got name: %s", name)
	}
}

func TestPumpNewContainerPump(t *testing.T) {
	config := &docker.Config{
		Tty: false,
	}
	container := &docker.Container{
		ID:     "8dfafdbc3a40",
		Config: config,
	}
	pump := newContainerPump(container, os.Stdout, os.Stderr)
	if pump == nil {
		t.Error("pump nil")
		return
	}
}

func TestPumpContainerPump(t *testing.T) {
	config := &docker.Config{
		Tty: true,
	}
	container := &docker.Container{
		ID:     "8dfafdbc3a40",
		Config: config,
	}
	pump := newContainerPump(container, os.Stdout, os.Stderr)
	logstream, route := make(chan *Message), &Route{}
	go func() {
		for msg := range logstream {
			t.Logf("message: %+v", msg)
		}
	}()
	pump.add(logstream, route)
	if pump.logstreams[logstream] != route {
		t.Error("expected pump to contain logstream matching route")
	}
	pump.send(&Message{Data: "test data"})

	pump.remove(logstream)
	if pump.logstreams[logstream] != nil {
		t.Error("logstream should have been removed")
	}
}

func TestPumpRoutingFrom(t *testing.T) {
	container := &docker.Container{
		ID: "8dfafdbc3a40",
	}
	p := &LogsPump{
		pumps:  make(map[string]*containerPump),
		routes: make(map[chan *update]struct{}),
	}

	if p.RoutingFrom(container.ID) != false {
		t.Errorf("expected RoutingFrom to return 'false'")
	}

	p.pumps[container.ID] = nil
	if p.RoutingFrom(container.ID) != true {
		t.Errorf("expected RoutingFrom to return 'true'")
	}
	if p.RoutingFrom("") != false {
		t.Errorf("expected RoutingFrom to return 'false'")
	}
	if p.RoutingFrom("foo") != false {
		t.Errorf("expected RoutingFrom to return 'false'")
	}
}
