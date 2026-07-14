package promise

import (
	"context"
	"fmt"
)

// ErrTimeout is returned from Await when a Promise wrapped by [Timeout] does not
// settle before its deadline. It wraps context.DeadlineExceeded, so both
// errors.Is(err, ErrTimeout) and errors.Is(err, context.DeadlineExceeded)
// report true.
var ErrTimeout = fmt.Errorf("promise: operation timed out: %w", context.DeadlineExceeded)

// ErrNoPromises is returned by combinators such as [Race] that cannot produce a
// meaningful result from an empty set of promises. JavaScript's Promise.race
// over an empty array never settles; blocking forever is not useful in Go, so
// the combinators return this error instead.
var ErrNoPromises = fmt.Errorf("promise: no promises provided")

// PanicError wraps a value recovered from a panic inside work started by [New]
// or [WithSignal]. It is returned from Await in place of the result, turning a
// panic — which would otherwise crash the process — into an ordinary Go error.
//
// If the recovered value is itself an error, [PanicError.Unwrap] exposes it, so
// errors.Is and errors.As see through the PanicError to the original.
type PanicError struct {
	// Value is the argument that was passed to panic.
	Value any
	// Stack is the goroutine stack trace captured at the moment of recovery.
	Stack []byte
}

// Error implements the error interface.
func (e *PanicError) Error() string {
	return fmt.Sprintf("promise: recovered panic: %v", e.Value)
}

// Unwrap returns the recovered value if it is itself an error, so errors.Is and
// errors.As can match against it. It returns nil otherwise.
func (e *PanicError) Unwrap() error {
	if err, ok := e.Value.(error); ok {
		return err
	}
	return nil
}
