package collections

import (
	"github.com/burrows99/async/abort"
	"github.com/burrows99/async/promise"
)

// Option configures [Map], [ForEach], and [NewQueue].
type Option func(*config)

type config struct {
	limit int // maximum concurrent tasks; <= 0 means unbounded
}

func configFrom(opts []Option) config {
	var c config
	for _, o := range opts {
		o(&c)
	}
	return c
}

// Concurrency bounds how many tasks run at once — the analogue of the
// { concurrency: n } option in p-map and p-queue. A value of zero or less means
// unbounded. It applies to [Map], [ForEach], and [NewQueue].
func Concurrency(n int) Option {
	return func(c *config) {
		c.limit = n
	}
}

// Map runs fn over every item concurrently and returns the results in input
// order — the analogue of `await Promise.all(items.map(fn))`, or of pMap when
// combined with [Concurrency].
//
// fn is an "async function": it returns a *[promise.Promise]. With
// [Concurrency], at most n promises are started at a time; without it, all start
// at once. Map is fail-fast like Promise.all: on the first error it stops
// starting new work and returns a nil slice with that error. In-flight work that
// ignores its signal runs to completion, exactly as it would in JavaScript.
func Map[T, U any](items []T, fn func(T) *promise.Promise[U], opts ...Option) ([]U, error) {
	n := len(items)
	results := make([]U, n)
	if n == 0 {
		return results, nil
	}

	limit := configFrom(opts).limit
	if limit <= 0 || limit > n {
		limit = n
	}

	// sem bounds concurrency; ctrl fans a single abort out to every task so a
	// failure stops queued work from starting.
	sem := make(chan struct{}, limit)
	ctrl := abort.NewController()
	errCh := make(chan error, n)

	for i, it := range items {
		go func() {
			select {
			case sem <- struct{}{}:
			case <-ctrl.Signal().Done():
				errCh <- ctrl.Signal().Reason()
				return
			}
			defer func() { <-sem }()

			if err := ctrl.Signal().ThrowIfAborted(); err != nil {
				errCh <- err
				return
			}
			v, err := fn(it).Await()
			if err == nil {
				results[i] = v
			}
			errCh <- err
		}()
	}

	var firstErr error
	for range items {
		if err := <-errCh; err != nil {
			firstErr = err
			ctrl.Abort() // stop queued tasks; stragglers drain into the buffered errCh
			break
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

// ForEach runs fn over every item concurrently for its side effects, discarding
// the results — the analogue of `await Promise.all(items.map(fn))` when you do
// not need the values. It honours [Concurrency] and is fail-fast like [Map].
func ForEach[T, U any](items []T, fn func(T) *promise.Promise[U], opts ...Option) error {
	_, err := Map(items, fn, opts...)
	return err
}
