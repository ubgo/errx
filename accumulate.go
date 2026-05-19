package errx

import (
	"fmt"
	"sync"
)

// Accumulator collects multiple failures instead of stopping at the first —
// the validation gap in Go. Each entry can carry a field path so a whole
// form/config/batch is validated in one pass. Goroutine-safe, so it doubles
// as the concurrent (parallel) accumulator.
//
//	acc := errx.NewAccumulator()
//	acc.Add("name", validateName(in.Name))
//	acc.Add("email", validateEmail(in.Email))
//	acc.AddErr(checkQuota(ctx))
//	return acc.ErrorOrNil() // nil, or one error listing every failure
type Accumulator struct {
	mu      sync.Mutex
	entries []accEntry
}

type accEntry struct {
	path string
	err  error
}

// NewAccumulator returns an empty accumulator.
func NewAccumulator() *Accumulator { return &Accumulator{} }

// Add records err under a field path (e.g. "items[2].price"). No-op if nil.
func (a *Accumulator) Add(path string, err error) {
	if err == nil {
		return
	}
	a.mu.Lock()
	a.entries = append(a.entries, accEntry{path: path, err: err})
	a.mu.Unlock()
}

// AddErr records err with no path. No-op if nil.
func (a *Accumulator) AddErr(err error) { a.Add("", err) }

// Len returns how many failures have been collected.
func (a *Accumulator) Len() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.entries)
}

// ErrorOrNil returns nil if nothing was collected, otherwise a single
// *Error (code "VALIDATION", ClassExpected) whose suppressed list holds
// every failure and whose fields hold the path → message map.
func (a *Accumulator) ErrorOrNil() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.entries) == 0 {
		return nil
	}
	e := &Error{
		msg:      fmt.Sprintf("%d validation error(s)", len(a.entries)),
		code:     "VALIDATION",
		class:    ClassExpected,
		severity: SevError,
		pcs:      callers(1),
	}
	for _, en := range a.entries {
		e.suppressed = append(e.suppressed, en.err)
		key := en.path
		if key == "" {
			key = "_"
		}
		e.fields = append(e.fields, Field{Key: key, Value: en.err.Error(), Safe: true})
	}
	return e
}

// Accumulate runs each fn fail-soft, collecting every error, and returns
// nil or one combined *Error. Use it for independent checks where you want
// all failures, not just the first.
func Accumulate(fns ...func() error) error {
	acc := NewAccumulator()
	for _, fn := range fns {
		acc.AddErr(fn())
	}
	return acc.ErrorOrNil()
}

// ParAccumulate runs every fn concurrently and collects all failures.
func ParAccumulate(fns ...func() error) error {
	acc := NewAccumulator()
	var wg sync.WaitGroup
	wg.Add(len(fns))
	for _, fn := range fns {
		go func(f func() error) {
			defer wg.Done()
			acc.AddErr(f())
		}(fn)
	}
	wg.Wait()
	return acc.ErrorOrNil()
}
