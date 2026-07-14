package collections_test

import (
	"testing"

	"github.com/burrows99/async/collections"
	"github.com/burrows99/async/promise"
)

func BenchmarkMapUnbounded(b *testing.B) {
	items := make([]int, 100)
	for range b.N {
		_, _ = collections.Map(items, func(i int) *promise.Promise[int] {
			return promise.New(func() (int, error) { return i, nil })
		})
	}
}

func BenchmarkMapConcurrency10(b *testing.B) {
	items := make([]int, 100)
	for range b.N {
		_, _ = collections.Map(items, func(i int) *promise.Promise[int] {
			return promise.New(func() (int, error) { return i, nil })
		}, collections.Concurrency(10))
	}
}

func BenchmarkQueue(b *testing.B) {
	for range b.N {
		q := collections.NewQueue[int](10)
		for j := range 100 {
			q.Add(func() (int, error) { return j, nil })
		}
		q.OnIdle()
	}
}
