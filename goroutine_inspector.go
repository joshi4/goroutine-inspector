package goroutine_inspector

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime/trace"
	"strconv"
	"sync"

	. "github.com/joshi4/goroutine-inspector/internal/trace"
)

type Trace struct {
	buf  *bytes.Buffer
	done sync.Once

	process          sync.Once
	leakedGoRoutines int
	stackTraces      []string
	err              error
}

// Start starts a trace for inspection.
//
// NOTE: Start must only be called once per executable
func Start() (*Trace, error) {
	t := &Trace{
		buf: new(bytes.Buffer),
	}

	if err := trace.Start(t.buf); err != nil {
		return nil, err
	}
	return t, nil
}

func shouldAddEvent(e *Event, fn string) bool {
	if fn == "" {
		return false
	}

	pattern := "github.com/joshi4/goroutine-inspector.Start|runtime/trace.Start|runtime.gcBgMark|runtime.addtimerLocked"
	ok, _ := regexp.MatchString(pattern, peekFn(e.Stk))
	return !ok
}

func peekFn(s []*Frame) string {
	if len(s) == 0 {
		return ""
	}
	return s[0].Fn
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
func (t *Trace) GoroutineLeaks() (int, []string, error) {
	t.Stop()

	t.process.Do(func() {
		count, info, err := goroutineLeaks(t.buf)
		t.leakedGoRoutines = count
		t.stackTraces = info
		t.err = err
	})
	return t.leakedGoRoutines, t.stackTraces, t.err
}

func GoroutineLeaksFromFile(filename string) (int, []string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return -1, nil, err
	}
	defer f.Close()
	return goroutineLeaks(f)
}

func (t *Trace) AssertGoroutineLeakCount(want int) error {
	count, leaks, err := t.GoroutineLeaks()
	if err != nil {
		return err
	}

	if count != want {
		return fmt.Errorf("goroutine_leak count mismatch: got = %d, expected %d, stack = %s", count, want, leaks)
	}
	return nil
}

func goroutineLeaks(r io.Reader) (int, []string, error) {
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
			if fn != nil && fn.Name != "" && shouldAddEvent(e, fn.Name) {
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
