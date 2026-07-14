package promise_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/burrows99/async/abort"
	"github.com/burrows99/async/promise"
)

func TestNewAwaitSuccess(t *testing.T) {
	t.Parallel()
	p := promise.New(func() (int, error) { return 42, nil })
	got, err := p.Await()
	if err != nil {
		t.Fatalf("Await returned error: %v", err)
	}
	if got != 42 {
		t.Fatalf("Await = %d, want 42", got)
	}
}

func TestNewAwaitError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("boom")
	p := promise.New(func() (int, error) { return 0, sentinel })
	got, err := p.Await()
	if !errors.Is(err, sentinel) {
		t.Fatalf("Await error = %v, want %v", err, sentinel)
	}
	if got != 0 {
		t.Fatalf("Await value = %d, want zero on error", got)
	}
}

// TestAwaitFreeFunction covers the positional `await p` spelling.
func TestAwaitFreeFunction(t *testing.T) {
	t.Parallel()
	p := promise.New(func() (string, error) { return "hi", nil })
	got, err := promise.Await(p)
	if err != nil || got != "hi" {
		t.Fatalf("promise.Await = (%q, %v), want (hi, nil)", got, err)
	}
}

func TestAwaitIsRepeatable(t *testing.T) {
	t.Parallel()
	p := promise.New(func() (string, error) { return "hi", nil })
	for i := range 3 {
		got, err := p.Await()
		if err != nil || got != "hi" {
			t.Fatalf("Await #%d = (%q, %v), want (hi, nil)", i, got, err)
		}
	}
}

func TestAwaitConcurrent(t *testing.T) {
	t.Parallel()
	p := promise.New(func() (int, error) { return 7, nil })
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := p.Await()
			if err != nil || got != 7 {
				t.Errorf("Await = (%d, %v), want (7, nil)", got, err)
			}
		}()
	}
	wg.Wait()
}

func TestPanicBecomesError(t *testing.T) {
	t.Parallel()
	p := promise.New(func() (int, error) { panic("kaboom") })
	got, err := p.Await()
	if got != 0 {
		t.Fatalf("value = %d, want zero after panic", got)
	}
	var pe *promise.PanicError
	if !errors.As(err, &pe) {
		t.Fatalf("error = %v, want *PanicError", err)
	}
	if pe.Value != "kaboom" {
		t.Fatalf("PanicError.Value = %v, want kaboom", pe.Value)
	}
	if len(pe.Stack) == 0 {
		t.Fatal("PanicError.Stack is empty, want a stack trace")
	}
	if !strings.Contains(pe.Error(), "kaboom") {
		t.Fatalf("PanicError.Error() = %q, want it to mention the panic value", pe.Error())
	}
}

// TestWithSignalAbortStopsTask verifies that Abort fires the signal and a
// well-behaved task stops with abort.ErrAborted.
func TestWithSignalAbortStopsTask(t *testing.T) {
	t.Parallel()
	p := promise.WithSignal(func(signal *abort.Signal) (int, error) {
		<-signal.Done()
		return 0, signal.Reason()
	})
	p.Abort()
	_, err := p.Await()
	if !errors.Is(err, abort.ErrAborted) {
		t.Fatalf("Await error = %v, want abort.ErrAborted", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ErrAborted must wrap context.Canceled, got %v", err)
	}
}

// TestPromiseSignalThreadsToOtherWork verifies a Promise's Signal can drive
// separate cancellable work.
func TestPromiseSignalThreadsToOtherWork(t *testing.T) {
	t.Parallel()
	p := promise.WithSignal(func(signal *abort.Signal) (int, error) {
		<-signal.Done()
		return 0, signal.Reason()
	})
	observed := make(chan struct{})
	go func() {
		<-p.Signal().Done() // same signal, observed elsewhere
		close(observed)
	}()
	p.Abort()
	select {
	case <-observed:
	case <-time.After(2 * time.Second):
		t.Fatal("Promise.Signal() did not fire for external observer")
	}
}

func TestAbortIsIdempotent(t *testing.T) {
	t.Parallel()
	p := promise.New(func() (int, error) { return 1, nil })
	p.Abort()
	p.Abort()
	_, _ = p.Await()
	p.Abort()
}

// TestExternalControllerAbortsTask shows the JS pattern: create a controller,
// let the task watch its signal, then abort the controller.
func TestExternalControllerAbortsTask(t *testing.T) {
	t.Parallel()
	c := abort.NewController()
	work := promise.New(func() (string, error) {
		select {
		case <-time.After(2 * time.Second):
			return "done", nil
		case <-c.Signal().Done():
			return "", c.Signal().Reason()
		}
	})
	c.Abort()
	_, err := work.Await()
	if !errors.Is(err, abort.ErrAborted) {
		t.Fatalf("Await error = %v, want abort.ErrAborted", err)
	}
}
