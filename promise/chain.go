package promise

import "github.com/burrows99/async/abort"

// settled returns a Promise that is already finished with (v, err), without
// starting a goroutine. It backs [Resolve] and [Reject].
func settled[T any](v T, err error) *Promise[T] {
	ctrl := abort.NewController()
	ctrl.Abort() // no task will run; release the controller's context immediately
	p := &Promise[T]{
		ctrl:  ctrl,
		done:  make(chan struct{}),
		value: v,
		err:   err,
	}
	close(p.done)
	return p
}

// Resolve returns a Promise already fulfilled with v — the analogue of
// JavaScript's Promise.resolve(value). It is handy for starting a chain or for
// returning a known value where a Promise is expected.
func Resolve[T any](v T) *Promise[T] {
	return settled(v, nil)
}

// Reject returns a Promise already rejected with err — the analogue of
// JavaScript's Promise.reject(reason). Await returns the zero value and err.
func Reject[T any](err error) *Promise[T] {
	var zero T
	return settled(zero, err)
}

// Then transforms a fulfilled Promise's value with fn and returns a new Promise
// for the result — the analogue of JavaScript's p.then(fn). Because Go methods
// cannot introduce new type parameters, Then is a package-level function, so
// chains nest inside-out rather than reading left to right:
//
//	// JS:  getUser(id).then(u => u.Name).then(greet)
//	promise.Then(promise.Then(getUser(id), userName), greet)
//
// If p rejects, Then propagates that error and does not call fn (a then with no
// rejection handler). fn may itself return an error, which rejects the new
// Promise. There is deliberately no Catch: handle errors the Go way, from the
// (T, error) that Await returns.
func Then[T, U any](p *Promise[T], fn func(T) (U, error)) *Promise[U] {
	return New(func() (U, error) {
		v, err := p.Await()
		if err != nil {
			var zero U
			return zero, err
		}
		return fn(v)
	})
}

// Finally schedules fn to run once the Promise settles, whatever the outcome,
// and returns a Promise that carries p's original result through unchanged — the
// analogue of p.finally(fn). fn takes no arguments and cannot alter the result;
// it is for cleanup only.
//
// In most Go code a plain defer is clearer and is what the docs recommend;
// Finally exists for symmetry with the JavaScript API.
func (p *Promise[T]) Finally(fn func()) *Promise[T] {
	return New(func() (T, error) {
		v, err := p.Await()
		fn()
		return v, err
	})
}
