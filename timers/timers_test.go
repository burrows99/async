package timers_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/burrows99/async/timers"
)

func TestSetTimeoutFires(t *testing.T) {
	t.Parallel()
	var fired atomic.Bool
	timers.SetTimeout(func() { fired.Store(true) }, 10*time.Millisecond)
	if fired.Load() {
		t.Fatal("fired immediately, want after the delay")
	}
	time.Sleep(40 * time.Millisecond)
	if !fired.Load() {
		t.Fatal("SetTimeout never fired")
	}
}

func TestClearTimeoutCancels(t *testing.T) {
	t.Parallel()
	var fired atomic.Bool
	h := timers.SetTimeout(func() { fired.Store(true) }, 30*time.Millisecond)
	timers.ClearTimeout(h)
	time.Sleep(60 * time.Millisecond)
	if fired.Load() {
		t.Fatal("ClearTimeout did not cancel the pending timeout")
	}
}

func TestClearTimeoutNilIsNoop(t *testing.T) {
	t.Parallel()
	timers.ClearTimeout(nil) // must not panic
}

func TestSetIntervalRepeatsUntilCleared(t *testing.T) {
	t.Parallel()
	var ticks atomic.Int64
	iv := timers.SetInterval(func() { ticks.Add(1) }, 10*time.Millisecond)
	time.Sleep(55 * time.Millisecond)
	timers.ClearInterval(iv)
	got := ticks.Load()
	if got < 3 {
		t.Fatalf("interval ticked %d times in ~55ms at 10ms, want >= 3", got)
	}

	// After clearing, ticking must stop.
	time.Sleep(40 * time.Millisecond)
	if after := ticks.Load(); after != got {
		t.Fatalf("interval kept ticking after ClearInterval: %d -> %d", got, after)
	}
}

func TestClearIntervalIdempotent(t *testing.T) {
	t.Parallel()
	iv := timers.SetInterval(func() {}, 10*time.Millisecond)
	timers.ClearInterval(iv)
	timers.ClearInterval(iv) // must not panic (double close guarded by sync.Once)
	timers.ClearInterval(nil)
}

func TestDebounceRunsOnceForABurst(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	debounced, _ := timers.Debounce(func() { calls.Add(1) }, 20*time.Millisecond)

	for range 5 {
		debounced()
		time.Sleep(2 * time.Millisecond) // rapid, all within the window
	}
	time.Sleep(50 * time.Millisecond)
	if got := calls.Load(); got != 1 {
		t.Fatalf("debounced ran %d times for a burst, want 1", got)
	}
}

func TestDebounceCancel(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	debounced, cancel := timers.Debounce(func() { calls.Add(1) }, 20*time.Millisecond)
	debounced()
	cancel()
	time.Sleep(40 * time.Millisecond)
	if got := calls.Load(); got != 0 {
		t.Fatalf("debounced ran %d times after cancel, want 0", got)
	}
}

func TestThrottleLeadingEdge(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	throttled, _ := timers.Throttle(func() { calls.Add(1) }, 50*time.Millisecond)

	// A burst inside one window should run exactly once (the leading call).
	for range 5 {
		throttled()
		time.Sleep(2 * time.Millisecond)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("throttled ran %d times in one window, want 1", got)
	}

	// After the window elapses, the next call runs again.
	time.Sleep(60 * time.Millisecond)
	throttled()
	if got := calls.Load(); got != 2 {
		t.Fatalf("throttled ran %d times total, want 2 after the window", got)
	}
}
