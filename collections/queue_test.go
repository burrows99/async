package collections_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/burrows99/async/collections"
	"github.com/burrows99/async/promise"
)

func TestQueueRunsAddedWork(t *testing.T) {
	t.Parallel()
	q := collections.NewQueue[int](2)
	p1 := q.Add(func() (int, error) { return 1, nil })
	p2 := q.Add(func() (int, error) { return 2, nil })

	v1, err1 := p1.Await()
	v2, err2 := p2.Await()
	if err1 != nil || err2 != nil {
		t.Fatalf("Await errors: %v, %v", err1, err2)
	}
	if v1 != 1 || v2 != 2 {
		t.Fatalf("results = (%d, %d), want (1, 2)", v1, v2)
	}
}

// TestQueueBoundsConcurrency verifies at most n tasks run at once.
func TestQueueBoundsConcurrency(t *testing.T) {
	t.Parallel()
	const limit = 3
	q := collections.NewQueue[struct{}](limit)
	var live, peak atomic.Int64

	ps := make([]*promise.Promise[struct{}], 20)
	for i := range ps {
		ps[i] = q.Add(func() (struct{}, error) {
			n := live.Add(1)
			for {
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			live.Add(-1)
			return struct{}{}, nil
		})
	}
	q.OnIdle()

	for _, p := range ps {
		if _, err := p.Await(); err != nil {
			t.Fatalf("task error: %v", err)
		}
	}
	if got := peak.Load(); got > limit {
		t.Fatalf("peak concurrency = %d, want <= %d", got, limit)
	}
	if peak.Load() == 0 {
		t.Fatal("peak concurrency = 0, tasks never ran")
	}
}

// TestQueueOnIdleWaitsForAll verifies OnIdle blocks until every task finishes.
func TestQueueOnIdleWaitsForAll(t *testing.T) {
	t.Parallel()
	q := collections.NewQueue[int](2)
	var done atomic.Int64
	for range 6 {
		q.Add(func() (int, error) {
			time.Sleep(10 * time.Millisecond)
			done.Add(1)
			return 0, nil
		})
	}
	q.OnIdle()
	if got := done.Load(); got != 6 {
		t.Fatalf("after OnIdle, %d tasks done, want 6", got)
	}
}

// TestQueuePanicBecomesError verifies a panic in an added task surfaces as a
// PanicError and does not leak a slot (OnIdle still returns, next task runs).
func TestQueuePanicBecomesError(t *testing.T) {
	t.Parallel()
	q := collections.NewQueue[int](1)
	p := q.Add(func() (int, error) { panic("worker boom") })
	next := q.Add(func() (int, error) { return 42, nil })

	_, err := p.Await()
	var pe *promise.PanicError
	if !errors.As(err, &pe) {
		t.Fatalf("panicked task error = %v, want *PanicError", err)
	}
	if v, err := next.Await(); err != nil || v != 42 {
		t.Fatalf("next task = (%d, %v), want (42, nil) — slot may have leaked", v, err)
	}
	q.OnIdle()
}

func TestNewQueueDefaultsToOne(t *testing.T) {
	t.Parallel()
	q := collections.NewQueue[int](0) // <1 becomes 1
	var live, peak atomic.Int64
	ps := make([]*promise.Promise[int], 4)
	for i := range ps {
		ps[i] = q.Add(func() (int, error) {
			n := live.Add(1)
			for {
				pk := peak.Load()
				if n <= pk || peak.CompareAndSwap(pk, n) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			live.Add(-1)
			return 0, nil
		})
	}
	q.OnIdle()
	for _, p := range ps {
		_, _ = p.Await()
	}
	if peak.Load() != 1 {
		t.Fatalf("peak concurrency = %d, want 1 for NewQueue(0)", peak.Load())
	}
}
