package collections_test

import (
	"errors"
	"testing"

	"github.com/burrows99/async/collections"
	"github.com/burrows99/async/internal/leaktest"
	"github.com/burrows99/async/promise"
)

// TestNoGoroutineLeak drives Map (bounded and unbounded, success and fail-fast)
// and Queue many times and asserts no goroutines are left behind.
func TestNoGoroutineLeak(t *testing.T) {
	leaktest.AssertNoGrowth(t, func() {
		fail := errors.New("x")
		items := []int{0, 1, 2, 3, 4, 5, 6, 7}

		for range 200 {
			// unbounded success
			_, _ = collections.Map(items, func(i int) *promise.Promise[int] {
				return promise.New(func() (int, error) { return i, nil })
			})

			// bounded success
			_, _ = collections.Map(items, func(i int) *promise.Promise[int] {
				return promise.New(func() (int, error) { return i, nil })
			}, collections.Concurrency(3))

			// bounded fail-fast
			_, _ = collections.Map(items, func(i int) *promise.Promise[int] {
				return promise.New(func() (int, error) {
					if i == 2 {
						return 0, fail
					}
					return i, nil
				})
			}, collections.Concurrency(2))

			// queue: submit, drain, await
			q := collections.NewQueue[int](3)
			ps := make([]*promise.Promise[int], len(items))
			for j := range items {
				ps[j] = q.Add(func() (int, error) { return j, nil })
			}
			q.OnIdle()
			for _, p := range ps {
				_, _ = p.Await()
			}
		}
	})
}
