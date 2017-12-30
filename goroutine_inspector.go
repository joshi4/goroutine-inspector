package goroutine_inspector

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime/trace"
	"strconv"
	"strings"
	"sync"

	. "github.com/joshi4/goroutine-inspector/internal/trace"
)

type Trace struct {
	buf       *bytes.Buffer
	whitelist []string
	done      sync.Once
}

var defaultWhitelist = []string{
	"runtime/trace.Start.func1",
	"testing.tRunner",
}

// Start starts a trace for inspection.
//
// NOTE: Start must only be called once per executable
func Start() (*Trace, error) {
	t := &Trace{
		buf:       new(bytes.Buffer),
		whitelist: defaultWhitelist,
	}

	t.buf.Reset()
	if err := trace.Start(t.buf); err != nil {
		return nil, err
	}
	return t, nil
}

// Stop stops the trace.
func (t *Trace) Stop() {
	t.done.Do(func() {
		trace.Stop()
	})
}

// GoroutineLeaks returns all go routines that were created
// but did not terminate during the trace period.
// GoroutineLeaks calls Stop()
func (t *Trace) GoroutineLeaks(whitelist ...string) error {
	t.Stop()
	whitelist = append(t.whitelist, whitelist...)
	return goroutineLeaks(t.buf, whitelist)
}

func GoroutineLeaksFromFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return goroutineLeaks(f, defaultWhitelist)
}

func goroutineLeaks(r io.Reader, whitelist []string) error {
	events, err := Parse(r, "")
	if err != nil {
		return err
	}

	gdesc := GoroutineStats(events)
	_ = gdesc

	stack := ""
	for _, e := range events {
		if fe, ok := hasGoroutineLeaked(e); ok {
			fn := gdesc[fe.G]
			if fn != nil && fn.Name != "" && !isWhitelisted(fn.Name, whitelist) {
				stack += printStack(e.Stk, fn.Name)
			}
		}
	}

	if stack == "" {
		return nil
	}
	return fmt.Errorf("%s", stack)
}

func isWhitelisted(name string, whitelist []string) bool {
	for _, wl := range whitelist {
		if strings.HasSuffix(name, wl) || strings.HasPrefix(name, "runtime.") {
			return true
		}
	}
	return false
}

func hasGoroutineLeaked(e *Event) (*Event, bool) {
	if e.Type != EvGoCreate {
		return nil, false
	}
	return traverseEventLinks(e)
}

func traverseEventLinks(e *Event) (*Event, bool) {
	if e.Link == nil {
		return e, (e.Type != EvGoEnd)
	}
	return traverseEventLinks(e.Link)
}

func printStack(s []*Frame, fn string) string {
	str := "call stack for:" + fn + "\n"
	for _, fr := range s {
		str += "function:" + fr.Fn + "\n" + "file:" + fr.File + "\nline:" + strconv.Itoa(fr.Line) + "\n"
	}
	return str + "end\n"
}
