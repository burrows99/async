package collections_test

import (
	"fmt"
	"sort"

	"github.com/burrows99/async/collections"
	"github.com/burrows99/async/promise"
)

// double is an "async function": doubling id, returned as a Promise.
func double(id int) *promise.Promise[int] {
	return promise.New(func() (int, error) {
		return id * 2, nil
	})
}

// The JavaScript this mirrors:
//
//	const doubled = await Promise.all(ids.map(double));
func ExampleMap() {
	doubled, err := collections.Map([]int{1, 2, 3, 4}, double)
	fmt.Println(doubled, err)
	// Output: [2 4 6 8] <nil>
}

// The JavaScript this mirrors:
//
//	const doubled = await pMap(ids, double, { concurrency: 2 });
func ExampleMap_concurrency() {
	doubled, err := collections.Map([]int{1, 2, 3, 4}, double, collections.Concurrency(2))
	fmt.Println(doubled, err)
	// Output: [2 4 6 8] <nil>
}

// The JavaScript this mirrors:
//
//	const queue = new PQueue({ concurrency: 2 });
//	jobs.forEach(j => queue.add(() => run(j)));
//	await queue.onIdle();
func ExampleQueue() {
	queue := collections.NewQueue[int](2)
	ps := make([]*promise.Promise[int], 3)
	for i := range ps {
		n := i + 1
		ps[i] = queue.Add(func() (int, error) { return n * n, nil })
	}
	queue.OnIdle()

	squares := make([]int, 0, len(ps))
	for _, p := range ps {
		v, _ := p.Await()
		squares = append(squares, v)
	}
	sort.Ints(squares)
	fmt.Println(squares)
	// Output: [1 4 9]
}
