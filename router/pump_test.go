package router

import (
	"os"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
)

func TestIgnoreContainer(t *testing.T) {
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

func TestLogsPumpName(t *testing.T) {
	p := &LogsPump{}
	if name := p.Name(); name != "pump" {
		t.Error("name should be 'pump' got:", name)
	}
}

func TestNewContainerPump(t *testing.T) {
	container := &docker.Container{
		ID: "8dfafdbc3a40",
	}
	pump := newContainerPump(container, os.Stdout, os.Stderr)
	if pump == nil {
		t.Error("pump nil")
		return
	}
}
func TestContainerPump(t *testing.T) {
	container := &docker.Container{
		ID: "8dfafdbc3a40",
	}
	pump := newContainerPump(container, os.Stdout, os.Stderr)
	logstream, route := make(chan *Message), &Route{}
	go func() {
		for msg := range logstream {
			t.Log("message:", msg)
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

func TestPumpSendTimeout(t *testing.T) {
	container := &docker.Container{
		ID: "8dfafdbc3a40",
	}
	pump := newContainerPump(container, os.Stdout, os.Stderr)
	ch, route := make(chan *Message), &Route{}
	pump.add(ch, route)
	pump.send(&Message{Data: "hellooo"})
	if pump.logstreams[ch] != nil {
		t.Error("expected logstream to be removed after timeout")
	}

}
