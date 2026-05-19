package errx

import "fmt"

// Recover converts a panic into a structured *Error (ClassDefect, SevFatal)
// and assigns it to *errp. Use it deferred so one bad input degrades to an
// error instead of crashing the process:
//
//	func handle() (err error) {
//		defer errx.Recover(&err)
//		mightPanic()
//		return nil
//	}
//
// If *errp already holds an error it is preserved as the cause.
func Recover(errp *error) {
	if r := recover(); r != nil {
		pe := fromPanic(r)
		if errp != nil && *errp != nil {
			pe.cause = *errp
		}
		if errp != nil {
			*errp = pe
		}
	}
}

// RecoverDo runs fn and returns a structured *Error if it panics.
func RecoverDo(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fromPanic(r)
		}
	}()
	return fn()
}

// Recovered reports whether err is a panic captured by this package.
func Recovered(err error) bool {
	e := Get(err)
	return e != nil && e.fromPanic
}

func fromPanic(r any) *Error {
	e := &Error{
		class:     ClassDefect,
		severity:  SevFatal,
		fromPanic: true,
		pcs:       callers(3), // skip fromPanic, the deferred closure, runtime
	}
	if err, ok := r.(error); ok {
		e.msg = "panic"
		e.cause = err
	} else {
		e.msg = fmt.Sprintf("panic: %v", r)
	}
	return e
}
