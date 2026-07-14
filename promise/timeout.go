package promise

import "time"

// Timeout returns a Promise that settles with p's result, or fails with
// [ErrTimeout] if p has not settled after d elapses. It is the analogue of
// racing work against AbortSignal.timeout(ms).
//
// On timeout the underlying promise is aborted via [Promise.Abort], so a task
// started with [WithSignal] stops promptly rather than lingering. Because
// ErrTimeout wraps context.DeadlineExceeded, callers can match either sentinel
// with errors.Is.
func Timeout[T any](p *Promise[T], d time.Duration) *Promise[T] {
	return New(func() (T, error) {
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-p.done:
			return p.value, p.err
		case <-timer.C:
			p.Abort()
			var zero T
			return zero, ErrTimeout
		}
	})
}
