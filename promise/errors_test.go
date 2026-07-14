package promise_test

import (
	"context"
	"errors"
	"testing"

	"github.com/burrows99/async/promise"
)

// TestPanicErrorUnwrapsErrorValue verifies that when code panics with an error
// value, errors.Is and errors.As can reach through the *PanicError to it.
func TestPanicErrorUnwrapsErrorValue(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("panicked with an error")
	p := promise.New(func() (int, error) {
		panic(sentinel)
	})
	_, err := p.Await()

	if !errors.Is(err, sentinel) {
		t.Fatalf("errors.Is could not see through PanicError to the panicked error: %v", err)
	}
	var pe *promise.PanicError
	if !errors.As(err, &pe) {
		t.Fatalf("errors.As could not extract *PanicError: %v", err)
	}
}

// TestPanicErrorUnwrapNonError verifies Unwrap returns nil for a non-error
// panic value.
func TestPanicErrorUnwrapNonError(t *testing.T) {
	t.Parallel()
	pe := &promise.PanicError{Value: "just a string"}
	if pe.Unwrap() != nil {
		t.Fatalf("Unwrap() = %v, want nil for a non-error panic value", pe.Unwrap())
	}
}

func TestErrTimeoutWrapsDeadlineExceeded(t *testing.T) {
	t.Parallel()
	if !errors.Is(promise.ErrTimeout, context.DeadlineExceeded) {
		t.Fatal("ErrTimeout must wrap context.DeadlineExceeded")
	}
}
