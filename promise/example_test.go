package promise_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/burrows99/async/abort"
	"github.com/burrows99/async/promise"
)

// The JavaScript this mirrors:
//
//	const p = doWork();
//	const result = await p;
func ExampleNew() {
	p := promise.New(func() (int, error) {
		return 2 + 2, nil
	})
	result, err := promise.Await(p)
	fmt.Println(result, err)
	// Output: 4 <nil>
}

// The JavaScript this mirrors:
//
//	const [a, b, c] = await Promise.all([taskA(), taskB(), taskC()]);
func ExampleAll() {
	sums, err := promise.All(
		promise.New(func() (int, error) { return 1, nil }),
		promise.New(func() (int, error) { return 2, nil }),
		promise.New(func() (int, error) { return 3, nil }),
	)
	fmt.Println(sums, err)
	// Output: [1 2 3] <nil>
}

// All2 joins promises of different types — the heterogeneous analogue of
// Promise.all that Go's generics cannot express as a single variadic call.
func ExampleAll2() {
	name, age, err := promise.All2(
		promise.New(func() (string, error) { return "ada", nil }),
		promise.New(func() (int, error) { return 36, nil }),
	)
	fmt.Println(name, age, err)
	// Output: ada 36 <nil>
}

// A panic inside a task becomes an ordinary error instead of crashing the
// process — the JavaScript "unhandled rejection is survivable" guarantee.
func ExamplePanicError() {
	p := promise.New(func() (int, error) {
		panic("something broke")
	})

	_, err := promise.Await(p)
	var pe *promise.PanicError
	if errors.As(err, &pe) {
		fmt.Println("recovered:", pe.Value)
	}
	// Output: recovered: something broke
}

// Timeout races work against a deadline, like AbortSignal.timeout(ms).
func ExampleTimeout() {
	slow := promise.WithSignal(func(signal *abort.Signal) (int, error) {
		select {
		case <-time.After(time.Hour):
			return 1, nil
		case <-signal.Done():
			return 0, signal.Reason()
		}
	})

	_, err := promise.Await(promise.Timeout(slow, 20*time.Millisecond))
	fmt.Println(errors.Is(err, promise.ErrTimeout))
	// Output: true
}
