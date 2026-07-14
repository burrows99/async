// Package abort provides Go analogues of the JavaScript AbortController and
// AbortSignal. A [Controller] hands out a [Signal] that cancellable work can
// observe; calling [Controller.Abort] fires the signal, exactly as
// controller.abort() does in the browser.
//
// The signal wraps a context.Context internally, so it bridges cleanly to the
// standard library: pass [Signal.Context] to any context-aware Go API and the
// underlying call is cancelled too.
package abort

import (
	"context"
	"fmt"
)

// ErrAborted is the reason a [Signal] reports once aborted — the analogue of the
// AbortError a JavaScript signal rejects with. It wraps context.Canceled, so
// both errors.Is(err, ErrAborted) and errors.Is(err, context.Canceled) report
// true.
var ErrAborted = fmt.Errorf("abort: aborted: %w", context.Canceled)

// Signal is observed by cancellable work to learn when it should stop. It is the
// Go analogue of a JavaScript AbortSignal — the object you pass to fetch as
// { signal }. Obtain one from a [Controller].
type Signal struct {
	ctx context.Context
}

// Aborted reports whether the signal has been aborted, mirroring the JavaScript
// signal.aborted property.
func (s *Signal) Aborted() bool {
	return s.ctx.Err() != nil
}

// Done returns a channel closed when the signal is aborted, for use in a select.
// It is the Go-idiomatic form of listening for the 'abort' event.
func (s *Signal) Done() <-chan struct{} {
	return s.ctx.Done()
}

// Reason returns [ErrAborted] once the signal is aborted, or nil before that. A
// task that stops on Done typically returns Reason as its error. It mirrors the
// JavaScript signal.reason property.
func (s *Signal) Reason() error {
	if s.ctx.Err() != nil {
		return ErrAborted
	}
	return nil
}

// ThrowIfAborted returns [ErrAborted] if the signal has been aborted and nil
// otherwise, mirroring JavaScript's signal.throwIfAborted(). Call it at a safe
// point to bail out of long CPU-bound work:
//
//	if err := signal.ThrowIfAborted(); err != nil {
//		return zero, err
//	}
func (s *Signal) ThrowIfAborted() error {
	if s.ctx.Err() != nil {
		return ErrAborted
	}
	return nil
}

// Context exposes the underlying context.Context, so the signal can be threaded
// into context-aware standard-library calls — the one place a context surfaces,
// just as a signal surfaces only at the fetch call in JavaScript.
func (s *Signal) Context() context.Context {
	return s.ctx
}

// Controller creates and aborts a [Signal], mirroring the JavaScript
// AbortController: hold the controller, hand its Signal to the work, and call
// Abort to cancel.
type Controller struct {
	signal *Signal
	cancel context.CancelFunc
}

// NewController returns a fresh [Controller] whose Signal is not yet aborted,
// mirroring new AbortController().
func NewController() *Controller {
	ctx, cancel := context.WithCancel(context.Background())
	return &Controller{
		signal: &Signal{ctx: ctx},
		cancel: cancel,
	}
}

// From returns a [Controller] whose Signal is also aborted when parent is done,
// so an outer signal cascades to inner work — the analogue of chaining an
// AbortSignal. Passing a nil parent behaves like [NewController].
func From(parent *Signal) *Controller {
	base := context.Background()
	if parent != nil {
		base = parent.ctx
	}
	ctx, cancel := context.WithCancel(base)
	return &Controller{
		signal: &Signal{ctx: ctx},
		cancel: cancel,
	}
}

// Signal returns the [Signal] this controller aborts.
func (c *Controller) Signal() *Signal {
	return c.signal
}

// Abort fires the controller's [Signal]. It is idempotent and safe to call from
// multiple goroutines — the analogue of controller.abort().
func (c *Controller) Abort() {
	c.cancel()
}
