package promise_test

import (
	"errors"
	"testing"
	"time"

	"github.com/burrows99/async/abort"
	"github.com/burrows99/async/promise"
)

func TestAnyFirstSuccessWins(t *testing.T) {
	t.Parallel()
	fail := promise.New(func() (string, error) { return "", errors.New("down") })
	slow := promise.WithSignal(func(signal *abort.Signal) (string, error) {
		select {
		case <-time.After(2 * time.Second):
			return "slow", nil
		case <-signal.Done():
			return "", signal.Reason()
		}
	})
	fast := promise.New(func() (string, error) { return "fast", nil })

	got, err := promise.Any(fail, slow, fast)
	if err != nil {
		t.Fatalf("Any error: %v", err)
	}
	if got != "fast" {
		t.Fatalf("Any = %q, want fast", got)
	}
}

func TestAnyAllFailAggregates(t *testing.T) {
	t.Parallel()
	e1 := errors.New("one")
	e2 := errors.New("two")
	e3 := errors.New("three")
	_, err := promise.Any(
		promise.New(func() (int, error) { return 0, e1 }),
		promise.New(func() (int, error) { return 0, e2 }),
		promise.New(func() (int, error) { return 0, e3 }),
	)

	var agg *promise.AggregateError
	if !errors.As(err, &agg) {
		t.Fatalf("Any error = %v, want *AggregateError", err)
	}
	if len(agg.Errors) != 3 {
		t.Fatalf("AggregateError has %d errors, want 3", len(agg.Errors))
	}
	// errors.Is should see through the aggregate to each reason.
	for _, want := range []error{e1, e2, e3} {
		if !errors.Is(err, want) {
			t.Fatalf("errors.Is could not find %v in the AggregateError", want)
		}
	}
}

func TestAnyEmpty(t *testing.T) {
	t.Parallel()
	_, err := promise.Any[int]()
	if !errors.Is(err, promise.ErrNoPromises) {
		t.Fatalf("Any() error = %v, want ErrNoPromises", err)
	}
}
