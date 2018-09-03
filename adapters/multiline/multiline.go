package multiline

import (
	"errors"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/gliderlabs/logspout/router"
)

const (
	matchFirst    = "first"
	matchLast     = "last"
	matchNonFirst = "nonfirst"
	matchNonLast  = "nonlast"
)

func init() {
	router.AdapterFactories.Register(NewMultilineAdapter, "multiline")
}

// Adapter collects multi-lint log entries and sends them to the next adapter as a single entry
type Adapter struct {
	out             chan *router.Message
	subAdapter      router.LogAdapter
	enableByDefault bool
	pattern         *regexp.Regexp
	separator       string
	matchFirstLine  bool
	negateMatch     bool
	flushAfter      time.Duration
	checkInterval   time.Duration
	buffers         map[string]*router.Message
	nextCheck       <-chan time.Time
}

// NewMultilineAdapter returns a configured multiline.Adapter
func NewMultilineAdapter(route *router.Route) (a router.LogAdapter, err error) {
	enableByDefault := true
	enableStr := os.Getenv("MULTILINE_ENABLE_DEFAULT")
	if enableStr != "" {
		var err error
		enableByDefault, err = strconv.ParseBool(enableStr)
		if err != nil {
			return nil, errors.New("multiline: invalid value for MULTILINE_ENABLE_DEFAULT (must be true|false): " + enableStr)
		}
	}

	pattern := os.Getenv("MULTILINE_PATTERN")
	if pattern == "" {
		pattern = `^\s`
	}

	separator := os.Getenv("MULTILINE_SEPARATOR")
	if separator == "" {
		separator = "\n"
	}
	patternRegexp, err := regexp.Compile(pattern)
	if err != nil {
		return nil, errors.New("multiline: invalid value for MULTILINE_PATTERN (must be regexp): " + pattern)
	}

	matchType := os.Getenv("MULTILINE_MATCH")
	if matchType == "" {
		matchType = matchNonFirst
	}
	matchType = strings.ToLower(matchType)
	matchFirstLine := false
	negateMatch := false
	switch matchType {
	case matchFirst:
		matchFirstLine = true
		negateMatch = false
	case matchLast:
		matchFirstLine = false
		negateMatch = false
	case matchNonFirst:
		matchFirstLine = true
		negateMatch = true
	case matchNonLast:
		matchFirstLine = false
		negateMatch = true
	default:
		return nil, errors.New("multiline: invalid value for MULTILINE_MATCH (must be one of first|last|nonfirst|nonlast): " + matchType)
	}

	flushAfter := 500 * time.Millisecond
	flushAfterStr := os.Getenv("MULTILINE_FLUSH_AFTER")
	if flushAfterStr != "" {
		timeoutMS, err := strconv.Atoi(flushAfterStr)
		if err != nil {
			return nil, errors.New("multiline: invalid value for multiline_timeout (must be number): " + flushAfterStr)
		}
		flushAfter = time.Duration(timeoutMS) * time.Millisecond
	}

	parts := strings.SplitN(route.Adapter, "+", 2)
	if len(parts) != 2 {
		return nil, errors.New("multiline: adapter must have a sub-adapter, eg: multiline+raw+tcp")
	}

	originalAdapter := route.Adapter
	route.Adapter = parts[1]
	factory, found := router.AdapterFactories.Lookup(route.AdapterType())
	if !found {
		return nil, errors.New("bad adapter: " + originalAdapter)
	}
	subAdapter, err := factory(route)
	if err != nil {
		return nil, err
	}
	route.Adapter = originalAdapter

	out := make(chan *router.Message)
	checkInterval := flushAfter / 2

	return &Adapter{
		out:             out,
		subAdapter:      subAdapter,
		enableByDefault: enableByDefault,
		pattern:         patternRegexp,
		separator:       separator,
		matchFirstLine:  matchFirstLine,
		negateMatch:     negateMatch,
		flushAfter:      flushAfter,
		checkInterval:   checkInterval,
		buffers:         make(map[string]*router.Message),
		nextCheck:       time.After(checkInterval),
	}, nil
}

// Stream sends log data to the next adapter
func (a *Adapter) Stream(logstream chan *router.Message) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		a.subAdapter.Stream(a.out)
		wg.Done()
	}()
	defer func() {
		for _, message := range a.buffers {
			a.out <- message
		}

		close(a.out)
		wg.Wait()
	}()

	for {
		select {
		case message, ok := <-logstream:
			if !ok {
				return
			}

			if !multilineContainer(message.Container, a.enableByDefault) {
				a.out <- message
				continue
			}

			cID := message.Container.ID
			old, oldExists := a.buffers[cID]
			if a.isFirstLine(message) {
				if oldExists {
					a.out <- old
				}

				a.buffers[cID] = message
			} else {
				isLastLine := a.isLastLine(message)
				
				if oldExists {
					old.Data += a.separator + message.Data
					message = old
				}

				if isLastLine {
					a.out <- message
					if oldExists {
						delete(a.buffers, cID)
					}
				} else {
					a.buffers[cID] = message
				}
			}
		case <-a.nextCheck:
			now := time.Now()

			for key, message := range a.buffers {
				if message.Time.Add(a.flushAfter).After(now) {
					a.out <- message
					delete(a.buffers, key)
				}
			}

			a.nextCheck = time.After(a.checkInterval)
		}
	}
}

func (a *Adapter) isFirstLine(message *router.Message) bool {
	if !a.matchFirstLine {
		return false
	}

	match := a.pattern.MatchString(message.Data)
	if a.negateMatch {
		return !match
	}

	return match
}

func (a *Adapter) isLastLine(message *router.Message) bool {
	if a.matchFirstLine {
		return false
	}

	match := a.pattern.MatchString(message.Data)
	if a.negateMatch {
		return !match
	}

	return match
}

func multilineContainer(container *docker.Container, def bool) bool {
	for _, kv := range container.Config.Env {
		kvp := strings.SplitN(kv, "=", 2)
		if len(kvp) == 2 && kvp[0] == "LOGSPOUT_MULTILINE" {
			switch strings.ToLower(kvp[1]) {
			case "true":
				return true
			case "false":
				return false
			}
			return def
		}
	}

	return def
}
