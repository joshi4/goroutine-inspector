package goroutine_inspector

import (
	"testing"
	"time"
)

func start(t *testing.T) *Trace {
	tr, err := Start()
	if err != nil {
		t.Error(err)
	}
	return tr
}

func TestGoroutineLeaks(t *testing.T) {
	tr := start(t)
	ch := make(chan bool)
	go routine(ch)
	<-ch

	// leak three go routines
	go routine(make(chan bool))
	go routine(make(chan bool))
	go routine(make(chan bool))

	if err := tr.GoroutineLeaks("routine"); err != nil {
		t.Error(err)
	}
}

func routine(ch chan bool) {
	ch <- false
}

func TestSleep(t *testing.T) {
	tr := start(t)
	time.Sleep(250 * time.Millisecond)
	if err := tr.GoroutineLeaks(); err != nil {
		t.Error(err)
	}
}

// TODO: rewrite this test
//func TestResponseBodyLeak(t *testing.T) {
//
//	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
//		io.WriteString(w, "okay")
//	})
//	s := httptest.NewServer(h)
//
//	tr := start(t)
//	defer func() {
//		if err := tr.AssertGoroutineLeakCount(1, "readLoop", "writeLoop", "net/http/httptest.(*Server).goServe.func1"); err != nil {
//			t.Error(err)
//		}
//	}()
//
//	cl := s.Client()
//	res, err := cl.Get(s.URL)
//	if err != nil {
//		t.Error(err)
//	}
//
//	// res.Body is not closed on purpose
//	b, err := ioutil.ReadAll(res.Body)
//	if err != nil {
//		t.Error(err)
//	}
//
//	if got, want := string(b), "okay"; got != want {
//		t.Errorf("unexpected response from test server: got=%s, want %s", got, want)
//	}
//}
