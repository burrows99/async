package collections

import (
	"sync"

	"github.com/burrows99/async/promise"
)

// Queue is a reusable, concurrency-limited work queue — the analogue of npm's
// p-queue (PQueue). Create one with [NewQueue], hand it work with [Queue.Add],
// and wait for everything to finish with [Queue.OnIdle].
//
// Unlike [Map], a Queue has no fixed set of items: add tasks over time, each
// returning its own [promise.Promise]. At most the configured number run
// concurrently; the rest wait their turn.
type Queue[T any] struct {
	sem chan struct{}
	wg  sync.WaitGroup
}

// NewQueue returns a [Queue] that runs at most concurrency tasks at once — the
// analogue of `new PQueue({ concurrency })`. A value of zero or less is treated
// as 1. A [Concurrency] option, if supplied, overrides the argument.
func NewQueue[T any](concurrency int, opts ...Option) *Queue[T] {
	if limit := configFrom(opts).limit; limit > 0 {
		concurrency = limit
	}
	if concurrency < 1 {
		concurrency = 1
	}
	return &Queue[T]{sem: make(chan struct{}, concurrency)}
}

// Add schedules fn on the queue and returns a [promise.Promise] for its result —
// the analogue of p-queue's queue.add(fn). The task waits for a free slot before
// running, so no more than the queue's concurrency run at once. The returned
// Promise settles like any other: Await it for the result, and a panic in fn
// becomes a [*promise.PanicError].
func (q *Queue[T]) Add(fn func() (T, error)) *promise.Promise[T] {
	q.wg.Add(1)
	return promise.New(func() (T, error) {
		defer q.wg.Done()
		q.sem <- struct{}{} // wait for a free slot
		defer func() { <-q.sem }()
		return fn()
	})
}

// OnIdle blocks until every task added so far has finished — the analogue of
// `await queue.onIdle()`. Adding more work after OnIdle returns is allowed;
// OnIdle simply waits for whatever has been added up to that point.
func (q *Queue[T]) OnIdle() {
	q.wg.Wait()
}
