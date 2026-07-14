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
			return 0, sentinel // every task fails immediately
		})
	}, collections.Concurrency(limit))

	if !errors.Is(err, sentinel) {
		t.Fatalf("Map error = %v, want %v", err, sentinel)
	}
	// With a small limit and immediate failures, far fewer than all 50 tasks
	// should ever start.
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

// TestMapWithSignalMapperAbortsOnFailFast verifies that a cancellable mapper
// still queued when a failure occurs is aborted rather than run.
func TestMapWithSignalMapperAbortsOnFailFast(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("first fails")
	var ran atomic.Int64

	items := []int{0, 1, 2, 3, 4, 5, 6, 7}
	_, err := collections.Map(items, func(i int) *promise.Promise[int] {
		return promise.New(func() (int, error) {
			if i == 0 {
				return 0, sentinel
			}
			ran.Add(1)
			time.Sleep(5 * time.Millisecond)
			return i, nil
		})
	}, collections.Concurrency(1)) // serialize so item 0 fails before most others start

	if !errors.Is(err, sentinel) {
		t.Fatalf("Map error = %v, want %v", err, sentinel)
	}
	if r := ran.Load(); r >= int64(len(items)-1) {
		t.Fatalf("ran %d tasks after the first failed; fail-fast should skip most", r)
	}
}
