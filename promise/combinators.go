package promise

import "sync"

// Result is the outcome of a single Promise as reported by [AllSettled]:
// exactly one of Value (on success) or Reason (on failure) is meaningful. The
// field names mirror the { value, reason } records returned by JavaScript's
// Promise.allSettled.
type Result[T any] struct {
	Value  T
	Reason error
}

// OK reports whether the Promise settled successfully, that is, Reason == nil.
func (r Result[T]) OK() bool {
	return r.Reason == nil
}

// All waits for every promise to settle and returns their values in input
// order. It is the analogue of JavaScript's Promise.all.
//
// On the first failure, All aborts the remaining sibling tasks and returns a nil
// slice with that error — Promise.all's fail-fast behaviour, plus the sibling
// cancellation raw JavaScript lacks (siblings started with [WithSignal] stop
// promptly; plain [New] siblings finish and have their results discarded). With
// no promises, All returns a non-nil empty slice and a nil error.
func All[T any](ps ...*Promise[T]) ([]T, error) {
	results := make([]T, len(ps))
	if len(ps) == 0 {
		return results, nil
	}
	errCh := make(chan error, len(ps))
	for i, p := range ps {
		go func() {
			v, err := p.Await()
			if err == nil {
				results[i] = v
			}
			errCh <- err
		}()
	}
	var firstErr error
	for range ps {
		if err := <-errCh; err != nil && firstErr == nil {
			firstErr = err
			for _, p := range ps {
				p.Abort()
			}
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

// AllSettled waits for every promise to settle and returns one [Result] per
// promise, in input order. It never returns early and never fails — the
// analogue of JavaScript's Promise.allSettled. Each Result carries either a
// Value (success) or a Reason (failure, including a [*PanicError] for a panicked
// task).
func AllSettled[T any](ps ...*Promise[T]) []Result[T] {
	results := make([]Result[T], len(ps))
	var wg sync.WaitGroup
	wg.Add(len(ps))
	for i, p := range ps {
		go func() {
			defer wg.Done()
			v, err := p.Await()
			results[i] = Result[T]{Value: v, Reason: err}
		}()
	}
	wg.Wait()
	return results
}

// Race waits for the first promise to settle — success or failure — and returns
// its result, mirroring JavaScript's Promise.race. Once one promise settles,
// Race aborts the rest. With no promises, Race returns [ErrNoPromises], since
// Promise.race over an empty array would hang forever.
func Race[T any](ps ...*Promise[T]) (T, error) {
	var zero T
	if len(ps) == 0 {
		return zero, ErrNoPromises
	}
	type outcome struct {
		value T
		err   error
	}
	// Buffered to len(ps) so losing goroutines never block on send after Race
	// has already returned with the winner.
	ch := make(chan outcome, len(ps))
	for _, p := range ps {
		go func() {
			v, err := p.Await()
			ch <- outcome{v, err}
		}()
	}
	o := <-ch
	for _, p := range ps {
		p.Abort()
	}
	return o.value, o.err
}
