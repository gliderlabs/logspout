package multiline

import (
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/gliderlabs/logspout/router"
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

type envTestData struct {
	env      []string
	def      bool
	expected bool
}

func (da *dummyAdapter) Stream(logstream chan *router.Message) {
	for m := range logstream {
		da.messages = append(da.messages, m)
	}
	da.Done()
}

func TestMultiline(t *testing.T) {
	tests := []*testData{
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
		{
			input: []string{
				"not yet",
				"Traceback",
				" tb1",
				" tb2!",
				"Error123",
				"no more traceback",
				"STATEMENT:",
				" still statement",
				"end of statement",
				"no more statement",
			},
			expected: []string{
				"not yet",
				"Traceback\n tb1\n tb2!\nError123",
				"no more traceback",
				"STATEMENT:\n still statement\nend of statement",
				"no more statement",
			},
			pattern:        regexp.MustCompile(`^(DETAIL:|STATEMENT:|Traceback|\s)`),
			matchFirstLine: false,
			negateMatch:    true,
		},
	}

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
			flushAfter:      time.Second * 10,
			checkInterval:   time.Millisecond * 100,
			buffers:         make(map[string]*router.Message),
			nextCheck:       time.After(time.Millisecond * 100),
			separator:       "\n",
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

func TestContainerEnv(t *testing.T) {
	tests := []envTestData{
		{
			def:      true,
			env:      []string{},
			expected: true,
		},
		{
			def:      false,
			env:      []string{},
			expected: false,
		},
		{
			def:      true,
			env:      []string{"LOGSPOUT_MULTILINE=true"},
			expected: true,
		},
		{
			def:      false,
			env:      []string{"LOGSPOUT_MULTILINE=true"},
			expected: true,
		},
		{
			def:      true,
			env:      []string{"LOGSPOUT_MULTILINE=false"},
			expected: false,
		},
		{
			def:      false,
			env:      []string{"LOGSPOUT_MULTILINE=false"},
			expected: false,
		},
	}

	for _, test := range tests {
		container := &docker.Container{
			ID: "test",
			Config: &docker.Config{
				Env: test.env,
			},
		}

		result := multilineContainer(container, test.def)

		if result != test.expected {
			t.Errorf("Expected: %v, Got: %v, env: %v", test.expected, result, test.env)
		}
	}
}

func replaceNewLines(str string) string {
	return strings.Replace(str, "\n", "\\n", -1)
}
