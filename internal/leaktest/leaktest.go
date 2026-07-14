// Package leaktest is a tiny, dependency-free goroutine-leak checker for the
// module's own tests. It deliberately avoids go.uber.org/goleak to keep the
// module's dependency set empty.
//
// The check is robust rather than exact: callers run a batch of work many times,
// so a per-operation leak accumulates far above the small tolerance for runtime
// noise, while a leak-free batch returns to baseline.
package leaktest

import (
	"runtime"
	"testing"
	"time"
)

// tolerance absorbs benign fluctuation in the background goroutine count
// (GC workers, the testing framework) that is unrelated to the code under test.
const tolerance = 2

// AssertNoGrowth records the goroutine count, runs fn, then fails t if the count
// does not settle back to (approximately) the starting level. Run the suspect
// operation many times inside fn so a real leak dwarfs the tolerance.
//
// The enclosing test must not call t.Parallel: goroutine counting cannot
// distinguish this test's goroutines from a sibling's.
func AssertNoGrowth(t *testing.T, fn func()) {
	t.Helper()
	base := stabilize()

	fn()

	for range 200 {
		runtime.GC()
		if runtime.NumGoroutine() <= base+tolerance {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("goroutine leak: baseline %d, still %d after work settled", base, runtime.NumGoroutine())
}

// stabilize waits for the goroutine count to stop changing and returns it, so
// transient startup goroutines don't inflate the baseline.
func stabilize() int {
	prev := -1
	for range 100 {
		runtime.GC()
		n := runtime.NumGoroutine()
		if n == prev {
			return n
		}
		prev = n
		time.Sleep(5 * time.Millisecond)
	}
	return runtime.NumGoroutine()
}
