package timers_test

import (
	"testing"
	"time"

	"github.com/burrows99/async/internal/leaktest"
	"github.com/burrows99/async/timers"
)

// TestNoGoroutineLeak drives every timer many times and asserts nothing is left
// running: cleared timeouts/intervals and cancelled debounces must not leak.
func TestNoGoroutineLeak(t *testing.T) {
	leaktest.AssertNoGrowth(t, func() {
		for range 300 {
			// SetTimeout cleared before it fires — no goroutine should spawn.
			to := timers.SetTimeout(func() {}, time.Hour)
			timers.ClearTimeout(to)

			// SetInterval must stop its goroutine on ClearInterval.
			iv := timers.SetInterval(func() {}, time.Hour)
			timers.ClearInterval(iv)

			// Debounce schedules an internal timer; cancel must drop it.
			deb, cancel := timers.Debounce(func() {}, time.Hour)
			deb()
			cancel()

			// Throttle (leading edge) leaves nothing pending.
			thr, _ := timers.Throttle(func() {}, time.Hour)
			thr()
		}
	})
}
