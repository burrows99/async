package collections_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/burrows99/async/collections"
	"github.com/burrows99/async/promise"
)

func TestMapPreservesOrder(t *testing.T) {
	t.Parallel()
	ids := []int{1, 2, 3, 4, 5}
	got, err := collections.Map(ids, func(id int) *promise.Promise[int] {
		return promise.New(func() (int, error) { return id * 10, nil })
	})
	if err != nil {
		t.Fatalf("Map error: %v", err)
	}
	want := []int{10, 20, 30, 40, 50}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Map = %v, want %v", got, want)
		}
	}
}

func TestMapEmpty(t *testing.T) {
	t.Parallel()
	got, err := collections.Map(nil, func(id int) *promise.Promise[int] {
		return promise.New(func() (int, error) { return id, nil })
	})
	if err != nil {
		t.Fatalf("Map(nil) error = %v, want nil", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("Map(nil) = %v, want non-nil empty slice", got)
	}
}

// TestMapWithLimitBoundsConcurrency verifies that no more than the limit run at
// once, by tracking the live count and its high-water mark.
func TestMapWithLimitBoundsConcurrency(t *testing.T) {
	t.Parallel()
	const limit = 3
	var live, peak atomic.Int64

	items := make([]int, 20)
	_, err := collections.Map(items, func(int) *promise.Promise[struct{}] {
		return promise.New(func() (struct{}, error) {
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
	}, collections.Concurrency(limit))
	if err != nil {
		t.Fatalf("Map error: %v", err)
	}
	if got := peak.Load(); got > limit {
		t.Fatalf("peak concurrency = %d, want <= %d", got, limit)
	}
	if got := peak.Load(); got == 0 {
		t.Fatal("peak concurrency = 0, tasks never ran")
	}
}

func TestMapFailFastStopsStartingWork(t *testing.T) {
	t.Parallel()
	const limit = 2
	sentinel := errors.New("boom")
	var started atomic.Int64

	items := make([]int, 50)
	_, err := collections.Map(items, func(i int) *promise.Promise[int] {
		return promise.New(func() (int, error) {
			started.Add(1)
			// A short delay so at most `limit` tasks are in flight when the
			// failure aborts the rest, giving a clear fail-fast window.
			time.Sleep(10 * time.Millisecond)
			return 0, sentinel
		})
	}, collections.Concurrency(limit))

	if !errors.Is(err, sentinel) {
		t.Fatalf("Map error = %v, want %v", err, sentinel)
	}
	// With a small limit, only a handful start before fail-fast stops the rest —
	// far fewer than all 50.
	if s := started.Load(); s >= int64(len(items)) {
		t.Fatalf("started %d tasks; fail-fast should have started fewer than %d", s, len(items))
	}
}

func TestForEachRunsSideEffects(t *testing.T) {
	t.Parallel()
	var sum atomic.Int64
	err := collections.ForEach([]int{1, 2, 3, 4}, func(n int) *promise.Promise[struct{}] {
		return promise.New(func() (struct{}, error) {
			sum.Add(int64(n))
			return struct{}{}, nil
		})
	})
	if err != nil {
		t.Fatalf("ForEach error: %v", err)
	}
	if sum.Load() != 10 {
		t.Fatalf("sum = %d, want 10", sum.Load())
	}
}

func TestForEachPropagatesError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("nope")
	err := collections.ForEach([]int{1, 2, 3}, func(n int) *promise.Promise[struct{}] {
		return promise.New(func() (struct{}, error) {
			if n == 2 {
				return struct{}{}, sentinel
			}
			return struct{}{}, nil
		})
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("ForEach error = %v, want %v", err, sentinel)
	}
}

// TestMapReturnsFailingError verifies Map surfaces the mapper's error, with no
// partial results, when exactly one item fails. It is deterministic because that
// error is the only one in play — no dependence on scheduling order.
func TestMapReturnsFailingError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("item 3 failed")
	got, err := collections.Map([]int{0, 1, 2, 3, 4, 5}, func(i int) *promise.Promise[int] {
		return promise.New(func() (int, error) {
			if i == 3 {
				return 0, sentinel
			}
			return i, nil
		})
	}, collections.Concurrency(2))

	if !errors.Is(err, sentinel) {
		t.Fatalf("Map error = %v, want %v", err, sentinel)
	}
	if got != nil {
		t.Fatalf("Map results = %v, want nil on error", got)
	}
}
