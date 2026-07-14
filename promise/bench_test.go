package promise_test

import (
	"sync"
	"testing"

	"github.com/burrows99/async/promise"
)

func BenchmarkNewAwait(b *testing.B) {
	for range b.N {
		p := promise.New(func() (int, error) { return 1, nil })
		_, _ = p.Await()
	}
}

func BenchmarkAll(b *testing.B) {
	for range b.N {
		_, _ = promise.All(
			promise.New(func() (int, error) { return 1, nil }),
			promise.New(func() (int, error) { return 2, nil }),
			promise.New(func() (int, error) { return 3, nil }),
		)
	}
}

func BenchmarkAll2(b *testing.B) {
	for range b.N {
		_, _, _ = promise.All2(
			promise.New(func() (int, error) { return 1, nil }),
			promise.New(func() (string, error) { return "x", nil }),
		)
	}
}

// BenchmarkRawGoroutineJoin is the hand-rolled stdlib equivalent of joining
// three concurrent tasks, as a baseline for the promise overhead above.
func BenchmarkRawGoroutineJoin(b *testing.B) {
	for range b.N {
		var wg sync.WaitGroup
		results := make([]int, 3)
		wg.Add(3)
		for i := range 3 {
			go func() {
				defer wg.Done()
				results[i] = i + 1
			}()
		}
		wg.Wait()
	}
}
