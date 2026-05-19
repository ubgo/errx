package errx

import (
	"errors"
	"fmt"
)

// New creates a leaf error with an operator message and an origin stack.
// Defaults: ClassExpected, SevError. Chain builders to enrich it:
//
//	return errx.New("db: deadlock on user_orders").
//		WithDomain("billing").WithCode("TX_RETRY").
//		WithPublic("Something went wrong, please retry").
//		WithRetryable(0)
func New(msg string) *Error {
	return &Error{
		msg:      msg,
		class:    ClassExpected,
		severity: SevError,
		pcs:      callers(1),
	}
}

// Newf is New with fmt.Sprintf formatting.
func Newf(format string, args ...any) *Error {
	return &Error{
		msg:      fmt.Sprintf(format, args...),
		class:    ClassExpected,
		severity: SevError,
		pcs:      callers(1),
	}
}

// Wrap annotates an existing error with an operator message while keeping
// the cause matchable via errors.Is/As (transparent by default; call
// Opaque() to make it a barrier). Returns nil if err is nil so it is safe
// at the tail of a function:
//
//	if err != nil { return errx.Wrap(err, "load config") }
//
// If err is already an *Error and msg is empty, Wrap inherits its identity
// instead of nesting redundantly.
func Wrap(err error, msg string) *Error {
	if err == nil {
		return nil
	}
	w := &Error{
		msg:      msg,
		cause:    err,
		class:    ClassExpected,
		severity: SevError,
		wrapPC:   wrapCaller(1),
		pcs:      callers(1),
	}
	// Inherit identity/class/severity from an inner *Error so the outer
	// annotation does not erase the precise inner classification.
	var inner *Error
	if errors.As(err, &inner) {
		w.domain = inner.domain
		w.code = inner.code
		w.class = inner.class
		w.severity = inner.severity
		w.retryable = inner.retryable
		w.retryAfter = inner.retryAfter
	}
	return w
}

// Wrapf is Wrap with fmt.Sprintf formatting.
func Wrapf(err error, format string, args ...any) *Error {
	if err == nil {
		return nil
	}
	w := &Error{
		msg:      fmt.Sprintf(format, args...),
		cause:    err,
		class:    ClassExpected,
		severity: SevError,
		wrapPC:   wrapCaller(1),
		pcs:      callers(1),
	}
	var inner *Error
	if errors.As(err, &inner) {
		w.domain = inner.domain
		w.code = inner.code
		w.class = inner.class
		w.severity = inner.severity
		w.retryable = inner.retryable
		w.retryAfter = inner.retryAfter
	}
	return w
}

// Note attaches a breadcrumb field to an existing error WITHOUT adding a
// wrapper frame — the answer to fmt.Errorf("...: %w") spam. If err is an
// *Error the note is appended in place and the same error is returned; for
// any other error a thin *Error wrapper carries the note.
//
//	errx.Note(err, "attempt", 3)
//	errx.Note(err, "tenant", tenantID)
func Note(err error, key string, value any) error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		e.fields = append(e.fields, Field{Key: key, Value: value})
		e.fp = ""
		return err
	}
	return &Error{
		cause:    err,
		class:    ClassExpected,
		severity: SevError,
		fields:   []Field{{Key: key, Value: value}},
		wrapPC:   wrapCaller(1),
		pcs:      callers(1),
	}
}

// Join bundles multiple errors into one, discarding nils. The result
// implements Unwrap() []error so errors.Is/As match any member. Returns nil
// if every input is nil, and the sole error unchanged if only one is non-nil.
func Join(errs ...error) error {
	n := 0
	var only error
	for _, e := range errs {
		if e != nil {
			n++
			only = e
		}
	}
	switch n {
	case 0:
		return nil
	case 1:
		return only
	}
	m := &multiError{errs: make([]error, 0, n)}
	for _, e := range errs {
		if e != nil {
			m.errs = append(m.errs, e)
		}
	}
	return m
}

type multiError struct {
	errs []error
}

func (m *multiError) Error() string {
	if len(m.errs) == 1 {
		return m.errs[0].Error()
	}
	b := make([]byte, 0, 64)
	b = append(b, fmt.Sprintf("%d errors:", len(m.errs))...)
	for _, e := range m.errs {
		b = append(b, "\n\t- "...)
		b = append(b, e.Error()...)
	}
	return string(b)
}

func (m *multiError) Unwrap() []error { return m.errs }
