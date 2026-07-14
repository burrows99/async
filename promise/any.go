package promise

import "fmt"

// AggregateError collects the rejection reasons from every promise, and is what
// [Any] returns when they all fail. It mirrors JavaScript's AggregateError,
// which Promise.any rejects with when no input fulfils.
//
// Its Unwrap returns all the underlying errors, so errors.Is and errors.As match
// against any of them.
type AggregateError struct {
	// Errors holds one entry per input promise, in input order.
	Errors []error
}

// Error implements the error interface.
func (e *AggregateError) Error() string {
	return fmt.Sprintf("promise: all %d promises were rejected", len(e.Errors))
}

// Unwrap returns every collected error, enabling errors.Is and errors.As to
// match against any of them (Go's multi-error unwrapping).
func (e *AggregateError) Unwrap() []error {
	return e.Errors
}

// Any waits for the first promise to fulfil and returns its value, mirroring
// JavaScript's Promise.any. As soon as one succeeds, Any aborts the rest.
//
// If every promise fails, Any returns a [*AggregateError] holding all the
// reasons in input order. With no promises, Any returns [ErrNoPromises].
func Any[T any](ps ...*Promise[T]) (T, error) {
	var zero T
	if len(ps) == 0 {
		return zero, ErrNoPromises
	}
	type outcome struct {
		value T
		err   error
		index int
	}
	ch := make(chan outcome, len(ps))
	for i, p := range ps {
		go func() {
			v, err := p.Await()
			ch <- outcome{v, err, i}
		}()
	}

	errs := make([]error, len(ps))
	for range ps {
		o := <-ch
		if o.err == nil {
			for _, p := range ps {
				p.Abort() // a winner settled; stop the losers
			}
			return o.value, nil
		}
		errs[o.index] = o.err
	}
	return zero, &AggregateError{Errors: errs}
}
