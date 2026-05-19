package errx

import (
	"context"
	"errors"
	"time"
)

// As is a generic, one-line errors.As: pull the first error of type T out
// of the chain without pre-declaring a variable.
//
//	if pe, ok := errx.As[*fs.PathError](err); ok { _ = pe.Path }
func As[T error](err error) (T, bool) {
	var t T
	if errors.As(err, &t) {
		return t, true
	}
	return t, false
}

// Get returns the first *Error in the chain, or nil.
func Get(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}

// Code returns the code of the first error in the chain carrying one —
// either an *errx.Error or any error exposing a `Code() string` method
// (so errxgen-generated and foreign typed errors are first-class without
// being errx.Error). Behavioral detection, segmentio-style.
func Code(err error) string {
	for err != nil {
		if e, ok := err.(*Error); ok && e.code != "" {
			return e.code
		}
		if c, ok := err.(interface{ Code() string }); ok {
			if v := c.Code(); v != "" {
				return v
			}
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
		case interface{ Unwrap() []error }:
			for _, c := range x.Unwrap() {
				if code := Code(c); code != "" {
					return code
				}
			}
			return ""
		default:
			return ""
		}
	}
	return ""
}

// HasCode reports whether any *Error in the chain has the given code.
func HasCode(err error, code string) bool {
	for err != nil {
		if e, ok := err.(*Error); ok && e.code == code {
			return true
		}
		if c, ok := err.(interface{ Code() string }); ok && c.Code() == code {
			return true
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
		case interface{ Unwrap() []error }:
			for _, c := range x.Unwrap() {
				if HasCode(c, code) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
	return false
}

// FindByCode returns the first *Error in the chain with the given code.
func FindByCode(err error, code string) *Error {
	for err != nil {
		if e, ok := err.(*Error); ok && e.code == code {
			return e
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
		case interface{ Unwrap() []error }:
			for _, c := range x.Unwrap() {
				if hit := FindByCode(c, code); hit != nil {
					return hit
				}
			}
			return nil
		default:
			return nil
		}
	}
	return nil
}

// ClassOf returns the class of the nearest *Error in the chain, or
// ClassCancelled if the chain contains context cancellation/deadline,
// else ClassExpected.
func ClassOf(err error) Class {
	if e := Get(err); e != nil {
		return e.class
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return ClassCancelled
	}
	return ClassExpected
}

// IsExpected / IsDefect / IsCancelled are convenience predicates over ClassOf.
func IsExpected(err error) bool  { return ClassOf(err) == ClassExpected }
func IsDefect(err error) bool    { return ClassOf(err) == ClassDefect }
func IsCancelled(err error) bool { return ClassOf(err) == ClassCancelled }

// IsRetryable reports whether the nearest *Error is marked retryable. It
// also treats context.DeadlineExceeded as non-retryable by default (the
// caller's deadline is gone), mirroring the AIP-194 stance.
func IsRetryable(err error) bool {
	if e := Get(err); e != nil {
		return e.retryable
	}
	return false
}

// RetryAfter returns the suggested retry delay from the nearest *Error.
func RetryAfter(err error) time.Duration {
	if e := Get(err); e != nil {
		return e.retryAfter
	}
	return 0
}
