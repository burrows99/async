package timers

import (
	"sync"
	"time"
)

// Timeout is the handle returned by [SetTimeout], the analogue of the timer id
// setTimeout returns in JavaScript. Pass it to [ClearTimeout] to cancel.
type Timeout struct {
	t *time.Timer
}

// SetTimeout runs fn once after d elapses, mirroring JavaScript's
// setTimeout(fn, ms). fn runs on its own goroutine. The returned [Timeout] can
// be passed to [ClearTimeout] to cancel before it fires.
func SetTimeout(fn func(), d time.Duration) *Timeout {
	return &Timeout{t: time.AfterFunc(d, fn)}
}

// ClearTimeout cancels a pending [Timeout] if it has not fired yet, mirroring
// clearTimeout(id). A nil handle is a no-op.
func ClearTimeout(t *Timeout) {
	if t != nil && t.t != nil {
		t.t.Stop()
	}
}

// Interval is the handle returned by [SetInterval]. Pass it to [ClearInterval]
// to stop the repeating calls.
type Interval struct {
	stop chan struct{}
	once sync.Once
}

// SetInterval calls fn every d, mirroring JavaScript's setInterval(fn, ms). The
// calls run on a dedicated goroutine, one after another; a slow fn delays the
// next tick rather than overlapping. Stop it with [ClearInterval].
func SetInterval(fn func(), d time.Duration) *Interval {
	iv := &Interval{stop: make(chan struct{})}
	go func() {
		ticker := time.NewTicker(d)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fn()
			case <-iv.stop:
				return
			}
		}
	}()
	return iv
}

// ClearInterval stops a running [Interval], mirroring clearInterval(id). It is
// idempotent, and a nil handle is a no-op.
func ClearInterval(iv *Interval) {
	if iv != nil {
		iv.once.Do(func() { close(iv.stop) })
	}
}

// Debounce returns a debounced wrapper around fn: each call to debounced
// postpones fn until d has passed with no further calls, so a burst of calls
// runs fn once, at the end. It mirrors lodash's _.debounce. cancel discards any
// pending call.
func Debounce(fn func(), d time.Duration) (debounced func(), cancel func()) {
	var mu sync.Mutex
	var timer *time.Timer

	debounced = func() {
		mu.Lock()
		defer mu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(d, fn)
	}
	cancel = func() {
		mu.Lock()
		defer mu.Unlock()
		if timer != nil {
			timer.Stop()
			timer = nil
		}
	}
	return debounced, cancel
}

// Throttle returns a throttled wrapper around fn that runs fn at most once per d
// on the leading edge: the first call runs immediately, and further calls within
// the same window are dropped. It mirrors lodash's _.throttle. cancel resets the
// window so the next call runs immediately.
func Throttle(fn func(), d time.Duration) (throttled func(), cancel func()) {
	var mu sync.Mutex
	var last time.Time

	throttled = func() {
		mu.Lock()
		now := time.Now()
		if last.IsZero() || now.Sub(last) >= d {
			last = now
			mu.Unlock()
			fn()
			return
		}
		mu.Unlock()
	}
	cancel = func() {
		mu.Lock()
		last = time.Time{}
		mu.Unlock()
	}
	return throttled, cancel
}
