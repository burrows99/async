package promise_test

import (
	"errors"
	"testing"
	"time"

	"github.com/burrows99/async/abort"
	"github.com/burrows99/async/internal/leaktest"
	"github.com/burrows99/async/promise"
)

// TestNoGoroutineLeak drives every promise path many times and asserts the
// goroutine count returns to baseline. Not parallel: it counts goroutines.
func TestNoGoroutineLeak(t *testing.T) {
	leaktest.AssertNoGrowth(t, func() {
		fail := errors.New("x")
		for range 300 {
			// success, error, and panic tasks
			okP := promise.New(func() (int, error) { return 1, nil })
			errP := promise.New(func() (int, error) { return 0, fail })
			panicP := promise.New(func() (int, error) { panic("boom") })
			_, _ = okP.Await()
			_, _ = errP.Await()
			_, _ = panicP.Await()

			// abandoned promise (never awaited) must still wind down
			_ = promise.New(func() (int, error) { return 7, nil })

			// combinators, including a WithSignal sibling aborted on failure
			sib := promise.WithSignal(func(s *abort.Signal) (int, error) {
				<-s.Done()
				return 0, s.Reason()
			})
			_, _ = promise.All(
				promise.New(func() (int, error) { return 0, fail }),
				sib,
			)

			_, _ = promise.Race(
				promise.New(func() (int, error) { return 1, nil }),
				promise.WithSignal(func(s *abort.Signal) (int, error) { <-s.Done(); return 0, s.Reason() }),
			)

			_, _ = promise.Any(
				promise.New(func() (int, error) { return 0, fail }),
				promise.New(func() (int, error) { return 2, nil }),
			)

			// chaining, timeout, retry, and explicit abort
			_, _ = promise.Await(promise.Then(promise.Resolve(1), func(n int) (int, error) { return n + 1, nil }))
			_, _ = promise.Await(promise.Timeout(promise.New(func() (int, error) { return 1, nil }), time.Second))
			_, _ = promise.Await(promise.Retry(func() (int, error) { return 1, nil }, promise.Attempts(2)))

			aborted := promise.WithSignal(func(s *abort.Signal) (int, error) { <-s.Done(); return 0, s.Reason() })
			aborted.Abort()
			_, _ = aborted.Await()
		}
	})
}
