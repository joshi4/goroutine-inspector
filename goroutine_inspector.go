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
	// start synOnce
	buf *bytes.Buffer

	done   sync.Once
	donech chan struct{}
	err    error
}

// Start starts a trace for inspection.
//
// NOTE: Start must only be called once per executable
func Start() (*Trace, error) {
	t := &Trace{
		buf:    new(bytes.Buffer),
		donech: make(chan struct{}),
	}

	if err := trace.Start(t.buf); err != nil {
		return nil, err
	}
	return t, nil
}

func shouldAddEvent(e *Event) bool {
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
		close(t.donech)
	})
}

// GoroutineLeaks returns all go routines that were created
// but did not terminate during the trace period.
// GoroutineLeaks calls Stop()
func (t *Trace) GoroutineLeaks() (int, []string, error) {
	t.Stop()
	return goroutineLeaks(t.buf)
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

	leakedGoRoutines := make(map[string]int)
	for _, e := range events {
		if e.Type == EvGoCreate && goroutineLeaked(e) {
			if shouldAddEvent(e) {
				leakedGoRoutines[printStack(e.Stk)] += 1
			}
		}
	}

	info := make([]string, 0, len(leakedGoRoutines))
	count := 0
	for k, v := range leakedGoRoutines {
		count += v
		info = append(info, fmt.Sprintf("count:%d\n stack:%s\n", v, k))
	}
	return count, info, nil
}

func goroutineLeaked(e *Event) bool {
	if e.Link == nil {
		return (e.Type != EvGoEnd)
	}
	return goroutineLeaked(e.Link)
}

func printStack(s []*Frame) string {
	str := ""
	for _, fr := range s {
		str += "function:" + fr.Fn + "\n" + "file:" + fr.File + "\nline:" + strconv.Itoa(fr.Line) + "\n"
	}
	return str + "end\n"
}
