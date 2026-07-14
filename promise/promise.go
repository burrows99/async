package promise

import (
	"runtime/debug"

	"github.com/burrows99/async/abort"
)

// Promise is a handle to the result of asynchronous work started with [New] or
// [WithSignal]. It is the Go analogue of a JavaScript Promise: a value that
// becomes available later, together with the error (if any) produced while
// computing it.
//
// A Promise is safe for concurrent use. The same Promise may be awaited from
// many goroutines; every caller observes the same settled (value, error) pair.
//
// Cancellation lives inside the Promise, which owns an [abort.Controller].
// Callers never pass a context: they call [Promise.Abort], and work that wants
// to observe cancellation opts in with [WithSignal].
type Promise[T any] struct {
	ctrl  *abort.Controller
	done  chan struct{}
	value T
	err   error
}

// aborter is the type-erased view of a Promise. Combinators use it to abort
// sibling tasks of differing element types through a single slice.
type aborter interface {
	Abort()
}

// newPromise is the shared engine behind New and WithSignal. Each Promise owns
// an [abort.Controller]; its Signal drives the task and its Abort is exposed as
// [Promise.Abort].
func newPromise[T any](run func(signal *abort.Signal) (T, error)) *Promise[T] {
	ctrl := abort.NewController()
	p := &Promise[T]{
		ctrl: ctrl,
		done: make(chan struct{}),
	}
	go func() {
		defer close(p.done)
		defer ctrl.Abort() // release the controller's context when the task settles
		defer func() {
			if r := recover(); r != nil {
				p.err = &PanicError{Value: r, Stack: debug.Stack()}
			}
		}()
		p.value, p.err = run(ctrl.Signal())
	}()
	return p
}

// New runs fn in a new goroutine and returns a [Promise] for its result — the Go
// analogue of calling an async function: `async function f() { ... }` yields a
// Promise the moment you call it.
//
// fn takes no signal. If fn panics, the panic is recovered and surfaced from
// Await as a [*PanicError]; a task New owns never tears down the process. Like a
// JavaScript async function with no signal wired in, this work cannot be
// interrupted mid-flight — [Promise.Abort] marks it aborted for awaiters, but fn
// runs to completion. When a task must stop on abort, use [WithSignal].
func New[T any](fn func() (T, error)) *Promise[T] {
	return newPromise(func(*abort.Signal) (T, error) {
		return fn()
	})
}

// WithSignal is [New] for work that wants to be cancellable. fn receives an
// [abort.Signal] it can observe — the Go analogue of an async function that
// accepts { signal } and passes it to fetch. When the Promise is aborted (by
// [Promise.Abort], by a combinator cancelling siblings, or by [Timeout]), the
// signal fires and a well-behaved fn returns promptly.
func WithSignal[T any](fn func(signal *abort.Signal) (T, error)) *Promise[T] {
	return newPromise(fn)
}

// Await blocks until the Promise settles and returns its (value, error). It is
// the method form of the [Await] function; both are the Go spelling of `await`.
// A settled Promise returns immediately; awaiting again re-returns the same
// result.
func (p *Promise[T]) Await() (T, error) {
	<-p.done
	return p.value, p.err
}

// Await blocks until p settles and returns its (value, error). It mirrors the
// JavaScript `await p` expression positionally — the operator wraps the promise.
func Await[T any](p *Promise[T]) (T, error) {
	return p.Await()
}

// Abort marks the Promise aborted — the analogue of AbortController.abort. It
// fires the [abort.Signal] handed to a [WithSignal] task, so a cancellable task
// stops promptly; a plain [New] task cannot be interrupted and runs to
// completion. Abort is idempotent and safe to call from multiple goroutines.
func (p *Promise[T]) Abort() {
	p.ctrl.Abort()
}

// Signal returns the Promise's own [abort.Signal], so the same cancellation can
// be threaded into other cancellable work.
func (p *Promise[T]) Signal() *abort.Signal {
	return p.ctrl.Signal()
}
