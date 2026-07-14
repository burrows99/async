package promise_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/burrows99/async/abort"
	"github.com/burrows99/async/promise"
)

func TestAllSuccessPreservesOrder(t *testing.T) {
	t.Parallel()
	// Later indices settle sooner, to prove ordering follows input, not timing.
	mk := func(v int, d time.Duration) *promise.Promise[int] {
		return promise.New(func() (int, error) {
			time.Sleep(d)
			return v, nil
		})
	}
	ps := []*promise.Promise[int]{
		mk(1, 30*time.Millisecond),
		mk(2, 10*time.Millisecond),
		mk(3, 1*time.Millisecond),
	}
	got, err := promise.All(ps...)
	if err != nil {
		t.Fatalf("All error: %v", err)
	}
	want := []int{1, 2, 3}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("All = %v, want %v", got, want)
		}
	}
}

func TestAllFailFastAbortsSiblings(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("fail")
	var siblingAborted atomic.Bool

	failing := promise.New(func() (int, error) {
		return 0, sentinel
	})
	sibling := promise.WithSignal(func(signal *abort.Signal) (int, error) {
		<-signal.Done() // aborted by All once failing errors
		siblingAborted.Store(true)
		return 0, signal.Reason()
	})

	_, err := promise.All(failing, sibling)
	if !errors.Is(err, sentinel) {
		t.Fatalf("All error = %v, want %v", err, sentinel)
	}
	if !siblingAborted.Load() {
		t.Fatal("sibling task was not aborted after first failure")
	}
}

func TestAllEmpty(t *testing.T) {
	t.Parallel()
	got, err := promise.All[int]()
	if err != nil {
		t.Fatalf("All() error = %v, want nil", err)
	}
	if got == nil {
		t.Fatal("All() = nil slice, want non-nil empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("All() len = %d, want 0", len(got))
	}
}

func TestAllSettledNeverFails(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("nope")
	ps := []*promise.Promise[int]{
		promise.New(func() (int, error) { return 10, nil }),
		promise.New(func() (int, error) { return 0, sentinel }),
		promise.New(func() (int, error) { panic("boom") }),
	}
	res := promise.AllSettled(ps...)
	if len(res) != 3 {
		t.Fatalf("AllSettled len = %d, want 3", len(res))
	}
	if !res[0].OK() || res[0].Value != 10 {
		t.Fatalf("res[0] = %+v, want ok with value 10", res[0])
	}
	if res[1].OK() || !errors.Is(res[1].Reason, sentinel) {
		t.Fatalf("res[1] = %+v, want failure with sentinel", res[1])
	}
	var pe *promise.PanicError
	if res[2].OK() || !errors.As(res[2].Reason, &pe) {
		t.Fatalf("res[2] = %+v, want a *PanicError", res[2])
	}
}

func TestRaceFirstToSettleWins(t *testing.T) {
	t.Parallel()
	fast := promise.New(func() (string, error) {
		return "fast", nil
	})
	slow := promise.WithSignal(func(signal *abort.Signal) (string, error) {
		select {
		case <-time.After(2 * time.Second):
			return "slow", nil
		case <-signal.Done():
			return "", signal.Reason()
		}
	})
	got, err := promise.Race(fast, slow)
	if err != nil {
		t.Fatalf("Race error: %v", err)
	}
	if got != "fast" {
		t.Fatalf("Race = %q, want fast", got)
	}
}

func TestRaceReturnsFirstError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("first")
	failFast := promise.New(func() (int, error) {
		return 0, sentinel
	})
	slow := promise.WithSignal(func(signal *abort.Signal) (int, error) {
		<-signal.Done()
		return 0, signal.Reason()
	})
	_, err := promise.Race(failFast, slow)
	if !errors.Is(err, sentinel) {
		t.Fatalf("Race error = %v, want %v (a rejection wins the race)", err, sentinel)
	}
}

func TestRaceEmpty(t *testing.T) {
	t.Parallel()
	_, err := promise.Race[int]()
	if !errors.Is(err, promise.ErrNoPromises) {
		t.Fatalf("Race() error = %v, want ErrNoPromises", err)
	}
}
