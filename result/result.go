// Package result is an OPTIONAL generics layer over errx for code that
// prefers a railway/Result style. It is never required: idiomatic
// (value, error) Go remains the primary interface of errx. Use this only
// where chaining genuinely reads better; Go culture resists pervasive
// monads, so keep it local.
package result

// Result holds either a value or an error.
type Result[T any] struct {
	val T
	err error
}

// Ok wraps a success value.
func Ok[T any](v T) Result[T] { return Result[T]{val: v} }

// Err wraps a failure.
func Err[T any](e error) Result[T] { return Result[T]{err: e} }

// From adapts an idiomatic (value, error) pair.
func From[T any](v T, err error) Result[T] {
	if err != nil {
		return Result[T]{err: err}
	}
	return Result[T]{val: v}
}

// Try runs fn and captures its outcome.
func Try[T any](fn func() (T, error)) Result[T] { return From(fn()) }

// IsOk reports success.
func (r Result[T]) IsOk() bool { return r.err == nil }

// IsErr reports failure.
func (r Result[T]) IsErr() bool { return r.err != nil }

// Err returns the error (nil on success).
func (r Result[T]) Err() error { return r.err }

// Value returns the value and ok=false on failure.
func (r Result[T]) Value() (T, bool) { return r.val, r.err == nil }

// Unwrap returns the value or panics with the error. Use only where a
// failure is genuinely unrecoverable.
func (r Result[T]) Unwrap() T {
	if r.err != nil {
		panic(r.err)
	}
	return r.val
}

// UnwrapOr returns the value or def on failure.
func (r Result[T]) UnwrapOr(def T) T {
	if r.err != nil {
		return def
	}
	return r.val
}

// UnwrapOrElse returns the value or the result of fn(err) on failure.
func (r Result[T]) UnwrapOrElse(fn func(error) T) T {
	if r.err != nil {
		return fn(r.err)
	}
	return r.val
}

// MapErr transforms the error side, leaving a success untouched.
func (r Result[T]) MapErr(fn func(error) error) Result[T] {
	if r.err != nil {
		return Result[T]{err: fn(r.err)}
	}
	return r
}

// Recover converts a failure into a success value.
func (r Result[T]) Recover(fn func(error) T) Result[T] {
	if r.err != nil {
		return Result[T]{val: fn(r.err)}
	}
	return r
}

// Match calls okFn on success or errFn on failure and returns its value.
func Match[T, R any](r Result[T], okFn func(T) R, errFn func(error) R) R {
	if r.err != nil {
		return errFn(r.err)
	}
	return okFn(r.val)
}

// Map transforms a success value, propagating a failure unchanged.
func Map[T, U any](r Result[T], fn func(T) U) Result[U] {
	if r.err != nil {
		return Result[U]{err: r.err}
	}
	return Result[U]{val: fn(r.val)}
}

// AndThen chains a fallible step (flatMap/bind).
func AndThen[T, U any](r Result[T], fn func(T) Result[U]) Result[U] {
	if r.err != nil {
		return Result[U]{err: r.err}
	}
	return fn(r.val)
}

// Collect is fail-fast: the first error short-circuits, otherwise all
// values are returned in order.
func Collect[T any](rs ...Result[T]) Result[[]T] {
	out := make([]T, 0, len(rs))
	for _, r := range rs {
		if r.err != nil {
			return Result[[]T]{err: r.err}
		}
		out = append(out, r.val)
	}
	return Result[[]T]{val: out}
}
