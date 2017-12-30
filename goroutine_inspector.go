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
func (t *Trace) GoroutineLeaks(whitelist []string) (int, []string, error) {
	t.Stop()
	whitelist = append(t.whitelist, whitelist...)
	return goroutineLeaks(t.buf, whitelist)
}

func GoroutineLeaksFromFile(filename string) (int, []string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return -1, nil, err
	}
	defer f.Close()
	return goroutineLeaks(f, defaultWhitelist)
}

func (t *Trace) AssertGoroutineLeakCount(want int, whitelist ...string) error {
	count, leaks, err := t.GoroutineLeaks(whitelist)
	if err != nil {
		return err
	}

	if count != want {
		return fmt.Errorf("goroutine_leak count mismatch: got = %d, expected %d, stack = %s", count, want, leaks)
	}
	return nil
}

func goroutineLeaks(r io.Reader, blacklist []string) (int, []string, error) {
	events, err := Parse(r, "")
	if err != nil {
		return -1, nil, err
	}

	gdesc := GoroutineStats(events)
	_ = gdesc

	leakedGoRoutines := make(map[string]int)
	for _, e := range events {
		if fe, ok := hasGoroutineLeaked(e); ok {
			fn := gdesc[fe.G]
			if fn != nil && fn.Name != "" && !isWhitelisted(fn.Name, blacklist) {
				leakedGoRoutines[printStack(e.Stk, fn.Name)] += 1
			}
		}
	}

	info := make([]string, 0, len(leakedGoRoutines))
	count := 0
	for k, v := range leakedGoRoutines {
		count += v
		info = append(info, fmt.Sprintf("count:%d\n call stack for:%s\n", v, k))
	}
	return count, info, nil
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
	str := fn + "\n"
	for _, fr := range s {
		str += "function:" + fr.Fn + "\n" + "file:" + fr.File + "\nline:" + strconv.Itoa(fr.Line) + "\n"
	}
	return str + "end\n"
}
