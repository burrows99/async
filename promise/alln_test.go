package promise_test

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/burrows99/async/abort"
	"github.com/burrows99/async/promise"
)

func TestAll2Heterogeneous(t *testing.T) {
	t.Parallel()
	nameP := promise.New(func() (string, error) { return "ada", nil })
	ageP := promise.New(func() (int, error) { return 36, nil })

	name, age, err := promise.All2(nameP, ageP)
	if err != nil {
		t.Fatalf("All2 error: %v", err)
	}
	if name != "ada" || age != 36 {
		t.Fatalf("All2 = (%q, %d), want (ada, 36)", name, age)
	}
}

func TestAll2FailFastAbortsSibling(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("bad")
	var aborted atomic.Bool

	bad := promise.New(func() (string, error) { return "", sentinel })
	sibling := promise.WithSignal(func(signal *abort.Signal) (int, error) {
		<-signal.Done()
		aborted.Store(true)
		return 0, signal.Reason()
	})

	name, age, err := promise.All2(bad, sibling)
	if !errors.Is(err, sentinel) {
		t.Fatalf("All2 error = %v, want %v", err, sentinel)
	}
	if name != "" || age != 0 {
		t.Fatalf("All2 = (%q, %d), want zero values on error", name, age)
	}
	if !aborted.Load() {
		t.Fatal("sibling was not aborted after first failure")
	}
}

// TestAllNMiddleArities exercises every generated arity between the two the
// other tests cover, so a per-arity indexing bug in the generator cannot hide.
func TestAllNMiddleArities(t *testing.T) {
	t.Parallel()
	mk := func(v int) *promise.Promise[int] {
		return promise.New(func() (int, error) { return v, nil })
	}

	t.Run("All3", func(t *testing.T) {
		t.Parallel()
		a, b, c, err := promise.All3(mk(1), mk(2), mk(3))
		if err != nil || a != 1 || b != 2 || c != 3 {
			t.Fatalf("All3 = (%d,%d,%d,%v)", a, b, c, err)
		}
	})
	t.Run("All4", func(t *testing.T) {
		t.Parallel()
		a, b, c, d, err := promise.All4(mk(1), mk(2), mk(3), mk(4))
		if err != nil || a != 1 || b != 2 || c != 3 || d != 4 {
			t.Fatalf("All4 = (%d,%d,%d,%d,%v)", a, b, c, d, err)
		}
	})
	t.Run("All5", func(t *testing.T) {
		t.Parallel()
		a, b, c, d, e, err := promise.All5(mk(1), mk(2), mk(3), mk(4), mk(5))
		if err != nil || a != 1 || b != 2 || c != 3 || d != 4 || e != 5 {
			t.Fatalf("All5 = (%d,%d,%d,%d,%d,%v)", a, b, c, d, e, err)
		}
	})
	t.Run("All6", func(t *testing.T) {
		t.Parallel()
		a, b, c, d, e, f, err := promise.All6(mk(1), mk(2), mk(3), mk(4), mk(5), mk(6))
		if err != nil || a != 1 || b != 2 || c != 3 || d != 4 || e != 5 || f != 6 {
			t.Fatalf("All6 = (%d,%d,%d,%d,%d,%d,%v)", a, b, c, d, e, f, err)
		}
	})
	t.Run("All7", func(t *testing.T) {
		t.Parallel()
		a, b, c, d, e, f, g, err := promise.All7(mk(1), mk(2), mk(3), mk(4), mk(5), mk(6), mk(7))
		if err != nil || a != 1 || b != 2 || c != 3 || d != 4 || e != 5 || f != 6 || g != 7 {
			t.Fatalf("All7 = (%d,%d,%d,%d,%d,%d,%d,%v)", a, b, c, d, e, f, g, err)
		}
	})
}

// TestAll8 exercises the widest generated arity end to end.
func TestAll8(t *testing.T) {
	t.Parallel()
	p1 := promise.New(func() (int, error) { return 1, nil })
	p2 := promise.New(func() (string, error) { return "2", nil })
	p3 := promise.New(func() (bool, error) { return true, nil })
	p4 := promise.New(func() (float64, error) { return 4.0, nil })
	p5 := promise.New(func() (int, error) { return 5, nil })
	p6 := promise.New(func() (rune, error) { return 'x', nil })
	p7 := promise.New(func() (byte, error) { return 7, nil })
	p8 := promise.New(func() ([]int, error) { return []int{8}, nil })

	v1, v2, v3, v4, v5, v6, v7, v8, err := promise.All8(p1, p2, p3, p4, p5, p6, p7, p8)
	if err != nil {
		t.Fatalf("All8 error: %v", err)
	}
	if v1 != 1 || v2 != "2" || !v3 || v4 != 4.0 || v5 != 5 || v6 != 'x' || v7 != 7 || len(v8) != 1 || v8[0] != 8 {
		t.Fatalf("All8 = (%d,%q,%t,%v,%d,%c,%d,%v), unexpected", v1, v2, v3, v4, v5, v6, v7, v8)
	}
}
