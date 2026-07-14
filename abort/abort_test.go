package abort_test

import (
	"context"
	"errors"
	"testing"

	"github.com/burrows99/async/abort"
)

func TestControllerAbortFiresSignal(t *testing.T) {
	t.Parallel()
	c := abort.NewController()
	sig := c.Signal()

	if sig.Aborted() {
		t.Fatal("signal aborted before Abort()")
	}
	if err := sig.ThrowIfAborted(); err != nil {
		t.Fatalf("ThrowIfAborted before abort = %v, want nil", err)
	}

	c.Abort()

	if !sig.Aborted() {
		t.Fatal("signal not aborted after Abort()")
	}
	if !errors.Is(sig.Reason(), abort.ErrAborted) {
		t.Fatalf("Reason() = %v, want ErrAborted", sig.Reason())
	}
	if !errors.Is(sig.ThrowIfAborted(), abort.ErrAborted) {
		t.Fatalf("ThrowIfAborted() = %v, want ErrAborted", sig.ThrowIfAborted())
	}

	select {
	case <-sig.Done():
	default:
		t.Fatal("Done() channel not closed after Abort()")
	}
}

func TestErrAbortedWrapsCanceled(t *testing.T) {
	t.Parallel()
	if !errors.Is(abort.ErrAborted, context.Canceled) {
		t.Fatal("ErrAborted must wrap context.Canceled")
	}
}

func TestSignalContextBridgesToStdlib(t *testing.T) {
	t.Parallel()
	c := abort.NewController()
	ctx := c.Signal().Context()
	if ctx.Err() != nil {
		t.Fatalf("Context().Err() = %v, want nil before abort", ctx.Err())
	}
	c.Abort()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("Context().Err() = %v, want context.Canceled after abort", ctx.Err())
	}
}

// TestFromCascades verifies that aborting a parent signal also aborts a child
// controller built with From.
func TestFromCascades(t *testing.T) {
	t.Parallel()
	parent := abort.NewController()
	child := abort.From(parent.Signal())

	parent.Abort()

	select {
	case <-child.Signal().Done():
	default:
		t.Fatal("child signal not aborted when parent aborted")
	}
}

func TestFromNilParent(t *testing.T) {
	t.Parallel()
	c := abort.From(nil)
	if c.Signal().Aborted() {
		t.Fatal("From(nil) signal should not be aborted")
	}
	c.Abort()
	if !c.Signal().Aborted() {
		t.Fatal("From(nil) signal should abort on Abort()")
	}
}

func TestAbortIdempotent(t *testing.T) {
	t.Parallel()
	c := abort.NewController()
	c.Abort()
	c.Abort() // must not panic
}
