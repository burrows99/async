package promise_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/burrows99/async/abort"
	"github.com/burrows99/async/promise"
)

func TestTimeoutFires(t *testing.T) {
	t.Parallel()
	var taskAborted atomic.Bool
	slow := promise.WithSignal(func(signal *abort.Signal) (int, error) {
		select {
		case <-time.After(2 * time.Second):
			return 1, nil
		case <-signal.Done():
			taskAborted.Store(true) // timeout should abort the underlying task
			return 0, signal.Reason()
		}
	})

	_, err := promise.Timeout(slow, 20*time.Millisecond).Await()
	if !errors.Is(err, promise.ErrTimeout) {
		t.Fatalf("Await error = %v, want ErrTimeout", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ErrTimeout does not wrap context.DeadlineExceeded: %v", err)
	}
	waitForTrue(t, &taskAborted, "underlying task was not aborted on timeout")
}

func TestTimeoutSucceedsWithinDeadline(t *testing.T) {
	t.Parallel()
	quick := promise.New(func() (int, error) {
		return 123, nil
	})
	got, err := promise.Timeout(quick, time.Second).Await()
	if err != nil {
		t.Fatalf("Await error = %v, want nil", err)
	}
	if got != 123 {
		t.Fatalf("Await = %d, want 123", got)
	}
}

func TestTimeoutPropagatesTaskError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("task failed")
	failing := promise.New(func() (int, error) {
		return 0, sentinel
	})
	_, err := promise.Timeout(failing, time.Second).Await()
	if !errors.Is(err, sentinel) {
		t.Fatalf("Await error = %v, want the task's own error %v", err, sentinel)
	}
}

func waitForTrue(t *testing.T, b *atomic.Bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if b.Load() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal(msg)
}
