package promise_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/burrows99/async/abort"
	"github.com/burrows99/async/promise"
)

func TestRetrySucceedsAfterFailures(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	p := promise.Retry(func() (string, error) {
		if calls.Add(1) < 3 {
			return "", errors.New("transient")
		}
		return "ok", nil
	}, promise.Attempts(5), promise.ConstantBackoff(time.Millisecond))

	got, err := p.Await()
	if err != nil {
		t.Fatalf("Retry error: %v", err)
	}
	if got != "ok" {
		t.Fatalf("Retry = %q, want ok", got)
	}
	if calls.Load() != 3 {
		t.Fatalf("called %d times, want 3 (2 failures then success)", calls.Load())
	}
}

func TestRetryGivesUpAfterAttempts(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("always")
	var calls atomic.Int64
	p := promise.Retry(func() (int, error) {
		calls.Add(1)
		return 0, sentinel
	}, promise.Attempts(3))

	_, err := p.Await()
	if !errors.Is(err, sentinel) {
		t.Fatalf("Retry error = %v, want the last error %v", err, sentinel)
	}
	if calls.Load() != 3 {
		t.Fatalf("called %d times, want exactly 3 attempts", calls.Load())
	}
}

func TestRetryDefaultsToThreeAttempts(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	p := promise.Retry(func() (int, error) {
		calls.Add(1)
		return 0, errors.New("no")
	})
	_, _ = p.Await()
	if calls.Load() != 3 {
		t.Fatalf("default attempts = %d, want 3", calls.Load())
	}
}

// TestRetryAbortStopsBackoff verifies aborting the retry promise interrupts the
// backoff wait rather than running the remaining attempts.
func TestRetryAbortStopsBackoff(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	p := promise.Retry(func() (int, error) {
		calls.Add(1)
		return 0, errors.New("fail")
	}, promise.Attempts(10), promise.ConstantBackoff(time.Second))

	// Let the first attempt run and enter its (1s) backoff, then abort.
	time.Sleep(50 * time.Millisecond)
	p.Abort()

	_, err := p.Await()
	if !errors.Is(err, abort.ErrAborted) {
		t.Fatalf("Retry error = %v, want abort.ErrAborted", err)
	}
	if c := calls.Load(); c > 2 {
		t.Fatalf("called %d times; abort should have stopped it near the first attempt", c)
	}
}
