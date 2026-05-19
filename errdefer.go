package errx

import (
	"errors"
	"io"
)

// OnError runs cleanup ONLY if *errp is non-nil when the enclosing
// function returns — the Zig `errdefer` semantics Go's plain `defer`
// cannot express without a hand-written `if *errp != nil` inside every
// deferred closure. Use it to roll back partial work on the failure path:
//
//	func write() (err error) {
//		f, err := os.Create(path)
//		if err != nil { return err }
//		defer errx.OnError(&err, func() { os.Remove(path) }) // undo on failure
//		defer f.Close()
//		return marshal(f)
//	}
func OnError(errp *error, cleanup func()) {
	if errp != nil && *errp != nil && cleanup != nil {
		cleanup()
	}
}

// OnSuccess runs fn ONLY if *errp is nil when the function returns.
func OnSuccess(errp *error, fn func()) {
	if errp != nil && *errp == nil && fn != nil {
		fn()
	}
}

// AppendSuppressed records extra as a secondary error on *errp without
// letting it mask the primary. If *errp is nil, extra becomes the error;
// if *errp is an *Error, extra is attached via Suppress; otherwise the
// pair is joined.
func AppendSuppressed(errp *error, extra error) {
	if errp == nil || extra == nil {
		return
	}
	switch {
	case *errp == nil:
		*errp = extra
	default:
		var e *Error
		if errors.As(*errp, &e) {
			e.Suppress(extra)
		} else {
			*errp = Join(*errp, extra)
		}
	}
}

// CloseSuppressing closes c and folds any close error into *errp as a
// SECONDARY error, so a failing Close never masks the real failure (the
// classic `defer f.Close()` bug Go has no stdlib remedy for):
//
//	defer errx.CloseSuppressing(&err, f)
func CloseSuppressing(errp *error, c io.Closer) {
	if c == nil {
		return
	}
	if cerr := c.Close(); cerr != nil {
		AppendSuppressed(errp, cerr)
	}
}
