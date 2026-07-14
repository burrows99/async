package promise

import (
	"time"

	"github.com/burrows99/async/abort"
)

// RetryOption configures [Retry].
type RetryOption func(*retryConfig)

type retryConfig struct {
	attempts int
	backoff  func(attempt int) time.Duration
}

// Attempts sets the total number of times [Retry] calls fn before giving up
// (the first try counts). Values below 1 are ignored. The default is 3.
func Attempts(n int) RetryOption {
	return func(c *retryConfig) {
		if n >= 1 {
			c.attempts = n
		}
	}
}

// ExpBackoff waits base after the first failure and doubles the wait after each
// subsequent one (base, 2·base, 4·base, …) — exponential backoff, as p-retry
// does by default.
func ExpBackoff(base time.Duration) RetryOption {
	return func(c *retryConfig) {
		c.backoff = func(attempt int) time.Duration {
			// cap the shift to guard against overflow on absurd attempt counts
			shift := min(attempt-1, 30)
			return base << shift
		}
	}
}

// ConstantBackoff waits the same duration d between every attempt.
func ConstantBackoff(d time.Duration) RetryOption {
	return func(c *retryConfig) {
		c.backoff = func(int) time.Duration { return d }
	}
}

// Retry runs fn, and if it returns an error, calls it again — up to [Attempts]
// times, waiting between tries per the chosen backoff. It returns a [Promise]
// that fulfils with the first success, or rejects with the last error if every
// attempt fails. It is the analogue of npm's p-retry.
//
// The returned Promise is cancellable: aborting it stops the retry loop, and
// the backoff wait is interrupted at once (an in-flight fn call, which Go cannot
// preempt, still finishes). By default fn is tried 3 times with no delay; pass
// [ExpBackoff] or [ConstantBackoff] to wait between tries.
func Retry[T any](fn func() (T, error), opts ...RetryOption) *Promise[T] {
	cfg := retryConfig{
		attempts: 3,
		backoff:  func(int) time.Duration { return 0 },
	}
	for _, o := range opts {
		o(&cfg)
	}

	return WithSignal(func(signal *abort.Signal) (T, error) {
		var zero T
		var lastErr error
		for attempt := 1; attempt <= cfg.attempts; attempt++ {
			if err := signal.ThrowIfAborted(); err != nil {
				return zero, err
			}
			v, err := fn()
			if err == nil {
				return v, nil
			}
			lastErr = err
			if attempt < cfg.attempts {
				select {
				case <-time.After(cfg.backoff(attempt)):
				case <-signal.Done():
					return zero, signal.Reason()
				}
			}
		}
		return zero, lastErr
	})
}
