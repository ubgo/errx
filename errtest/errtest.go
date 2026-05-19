// Package errtest provides assertion helpers for testing errx-based errors:
// check a code, class, retryability, or chain membership without
// hand-rolling errors.As plumbing in every test.
package errtest

import (
	"errors"

	"github.com/ubgo/errx"
)

// TB is the subset of testing.TB the helpers need (also satisfied by
// testing.T and testing.B).
type TB interface {
	Helper()
	Fatalf(format string, args ...any)
}

// Code asserts that err carries the given code somewhere in its chain.
func Code(t TB, err error, want string) {
	t.Helper()
	if got := errx.Code(err); got != want {
		t.Fatalf("errx.Code = %q, want %q (err: %v)", got, want, err)
	}
}

// Class asserts the resolved class of err.
func Class(t TB, err error, want errx.Class) {
	t.Helper()
	if got := errx.ClassOf(err); got != want {
		t.Fatalf("errx.ClassOf = %s, want %s (err: %v)", got, want, err)
	}
}

// Retryable asserts whether err is marked retryable.
func Retryable(t TB, err error, want bool) {
	t.Helper()
	if got := errx.IsRetryable(err); got != want {
		t.Fatalf("errx.IsRetryable = %v, want %v (err: %v)", got, want, err)
	}
}

// InChain asserts that target is matchable via errors.Is from err.
func InChain(t TB, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("errors.Is(%v, %v) = false, want true", err, target)
	}
}

// HasCode asserts that some *errx.Error in the chain has the code.
func HasCode(t TB, err error, code string) {
	t.Helper()
	if !errx.HasCode(err, code) {
		t.Fatalf("expected code %q somewhere in chain, err: %v", code, err)
	}
}

// NoError fails the test if err is non-nil.
func NoError(t TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Fingerprint asserts that two errors group to the same fingerprint
// (the same logical bug), regardless of their variable messages.
func Fingerprint(t TB, a, b error) {
	t.Helper()
	fa, fb := errx.Fingerprint(a), errx.Fingerprint(b)
	if fa == "" || fa != fb {
		t.Fatalf("fingerprints differ: %q vs %q", fa, fb)
	}
}
