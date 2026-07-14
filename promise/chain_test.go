package promise_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/burrows99/async/promise"
)

func TestResolve(t *testing.T) {
	t.Parallel()
	got, err := promise.Resolve(42).Await()
	if err != nil || got != 42 {
		t.Fatalf("Resolve(42).Await() = (%d, %v), want (42, nil)", got, err)
	}
}

func TestReject(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("nope")
	got, err := promise.Reject[int](sentinel).Await()
	if !errors.Is(err, sentinel) {
		t.Fatalf("Reject error = %v, want %v", err, sentinel)
	}
	if got != 0 {
		t.Fatalf("Reject value = %d, want zero", got)
	}
}

func TestThenTransforms(t *testing.T) {
	t.Parallel()
	// A cross-type transform: int -> string.
	label, err := promise.Then(promise.Resolve(42), func(n int) (string, error) {
		return fmt.Sprintf("n=%d", n), nil
	}).Await()
	if err != nil || label != "n=42" {
		t.Fatalf("Then int->string = (%q, %v), want (n=42, nil)", label, err)
	}
}

func TestThenChains(t *testing.T) {
	t.Parallel()
	// JS: Resolve(3).then(n => n+1).then(n => n*10)
	p := promise.Then(
		promise.Then(promise.Resolve(3), func(n int) (int, error) { return n + 1, nil }),
		func(n int) (int, error) { return n * 10, nil },
	)
	got, err := p.Await()
	if err != nil || got != 40 {
		t.Fatalf("chained Then = (%d, %v), want (40, nil)", got, err)
	}
}

func TestThenSkipsOnRejection(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("upstream failed")
	called := false
	p := promise.Then(promise.Reject[int](sentinel), func(n int) (int, error) {
		called = true
		return n, nil
	})
	_, err := p.Await()
	if !errors.Is(err, sentinel) {
		t.Fatalf("Then error = %v, want propagated %v", err, sentinel)
	}
	if called {
		t.Fatal("Then called fn even though the upstream promise rejected")
	}
}

func TestThenFnErrorRejects(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("transform failed")
	p := promise.Then(promise.Resolve(1), func(int) (int, error) {
		return 0, sentinel
	})
	if _, err := p.Await(); !errors.Is(err, sentinel) {
		t.Fatalf("Then error = %v, want %v from the transform", err, sentinel)
	}
}

func TestFinallyRunsOnSuccess(t *testing.T) {
	t.Parallel()
	ran := false
	got, err := promise.Resolve("v").Finally(func() { ran = true }).Await()
	if err != nil || got != "v" {
		t.Fatalf("Finally passthrough = (%q, %v), want (v, nil)", got, err)
	}
	if !ran {
		t.Fatal("Finally fn did not run on success")
	}
}

func TestFinallyRunsOnFailure(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("boom")
	ran := false
	_, err := promise.Reject[int](sentinel).Finally(func() { ran = true }).Await()
	if !errors.Is(err, sentinel) {
		t.Fatalf("Finally error = %v, want %v passed through", err, sentinel)
	}
	if !ran {
		t.Fatal("Finally fn did not run on failure")
	}
}
