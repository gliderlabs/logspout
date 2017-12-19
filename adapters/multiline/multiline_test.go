package multiline

import (
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/gliderlabs/logspout/router"
	"strings"
)

type dummyAdapter struct {
	messages []*router.Message
	*sync.WaitGroup
}

type testData struct {
	input          []string
	expected       []string
	pattern        *regexp.Regexp
	matchFirstLine bool
	negateMatch    bool
}

func (da *dummyAdapter) Stream(logstream chan *router.Message) {
	for m := range logstream {
		da.messages = append(da.messages, m)
	}
	da.Done()
}

var tests = []*testData{
	{
		input: []string{
			"some",
			"  multi",
			"  line",
			"other",
			"  multiline",
		},
		expected: []string{
			"some\n  multi\n  line",
			"other\n  multiline",
		},
		pattern:        regexp.MustCompile(`^\s`),
		matchFirstLine: true,
		negateMatch:    true,
	},
	{
		input: []string{
			"some:",
			"multi",
			"line",
			"other:",
			"multiline",
		},
		expected: []string{
			"some:\nmulti\nline",
			"other:\nmultiline",
		},
		pattern:        regexp.MustCompile(`:$`),
		matchFirstLine: true,
		negateMatch:    false,
	},
	{
		input: []string{
			"some$",
			"multi$",
			"line",
			"other$",
			"multiline",
		},
		expected: []string{
			"some$\nmulti$\nline",
			"other$\nmultiline",
		},
		pattern:        regexp.MustCompile(`\$$`),
		matchFirstLine: false,
		negateMatch:    true,
	},
	{
		input: []string{
			"some",
			"multi",
			"line!",
			"other",
			"multiline!",
		},
		expected: []string{
			"some\nmulti\nline!",
			"other\nmultiline!",
		},
		pattern:        regexp.MustCompile(`!$`),
		matchFirstLine: false,
		negateMatch:    false,
	},
}

func TestMultiline(t *testing.T) {
	for _, test := range tests {
		in := make(chan *router.Message)
		out := make(chan *router.Message)
		container := &docker.Container{
			ID:     "test",
			Config: &docker.Config{},
		}

		da := &dummyAdapter{make([]*router.Message, 0), &sync.WaitGroup{}}
		da.Add(1)

		ma := &Adapter{
			out:             out,
			subAdapter:      da,
			enableByDefault: true,
			pattern:         test.pattern,
			matchFirstLine:  test.matchFirstLine,
			negateMatch:     test.negateMatch,
			flushAfter:      time.Millisecond * 200,
			checkInterval:   time.Microsecond * 100,
			buffers:         make(map[string]*router.Message),
			nextCheck:       time.After(time.Microsecond * 100),
		}

		go ma.Stream(in)

		for _, i := range test.input {
			in <- &router.Message{
				Container: container,
				Data:      i,
				Source:    "stdout",
				Time:      time.Now(),
			}
		}

		close(in)
		da.Wait()

		for i, m := range da.messages {
			if m.Data != test.expected[i] {
				t.Errorf("Expected: '%v', Got: '%v'", replaceNewLines(test.expected[i]), replaceNewLines(m.Data))
			}
		}
	}
}

func replaceNewLines(str string) string {
	return strings.Replace(str, "\n", "\\n", -1)
}