package promise_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/burrows99/async/promise"
)

// FuzzAll drives All with arbitrary success/failure patterns (one byte per
// promise; odd = fail) and checks the invariants under -race: success returns
// every value in order, and any failure returns a nil slice with an error.
func FuzzAll(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{1})
	f.Add([]byte{0, 1, 0, 1, 0})
	f.Add([]byte{2, 4, 6, 8})

	f.Fuzz(func(t *testing.T, spec []byte) {
		if len(spec) == 0 || len(spec) > 32 {
			t.Skip()
		}
		anyFail := false
		ps := make([]*promise.Promise[int], len(spec))
		for i := range spec {
			fails := spec[i]&1 == 1
			anyFail = anyFail || fails
			ps[i] = promise.New(func() (int, error) {
				if fails {
					return 0, fmt.Errorf("fail %d", i)
				}
				return i, nil
			})
		}

		got, err := promise.All(ps...)
		if anyFail {
			if err == nil {
				t.Fatalf("spec %v: want an error", spec)
			}
			if got != nil {
				t.Fatalf("spec %v: want nil results on error, got %v", spec, got)
			}
			return
		}
		if err != nil {
			t.Fatalf("spec %v: unexpected error %v", spec, err)
		}
		if len(got) != len(spec) {
			t.Fatalf("spec %v: len(got)=%d, want %d", spec, len(got), len(spec))
		}
		for i := range got {
			if got[i] != i {
				t.Fatalf("spec %v: got[%d]=%d, want %d", spec, i, got[i], i)
			}
		}
	})
}

// FuzzAny drives Any with arbitrary patterns and checks: if all fail it returns
// an AggregateError holding every reason; otherwise it returns the value of a
// promise that actually succeeded.
func FuzzAny(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{1, 1, 1})
	f.Add([]byte{1, 0, 1})

	f.Fuzz(func(t *testing.T, spec []byte) {
		if len(spec) == 0 || len(spec) > 32 {
			t.Skip()
		}
		allFail := true
		ps := make([]*promise.Promise[int], len(spec))
		for i := range spec {
			fails := spec[i]&1 == 1
			allFail = allFail && fails
			ps[i] = promise.New(func() (int, error) {
				if fails {
					return 0, fmt.Errorf("fail %d", i)
				}
				return i, nil
			})
		}

		got, err := promise.Any(ps...)
		if allFail {
			var agg *promise.AggregateError
			if !errors.As(err, &agg) {
				t.Fatalf("spec %v: want *AggregateError, got %v", spec, err)
			}
			if len(agg.Errors) != len(spec) {
				t.Fatalf("spec %v: aggregate has %d errors, want %d", spec, len(agg.Errors), len(spec))
			}
			return
		}
		if err != nil {
			t.Fatalf("spec %v: unexpected error %v", spec, err)
		}
		if got < 0 || got >= len(spec) || spec[got]&1 == 1 {
			t.Fatalf("spec %v: Any returned %d, which was not a succeeding promise", spec, got)
		}
	})
}
