package errx_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ubgo/errx"
	"github.com/ubgo/errx/errtest"
)

func TestRecoverFromValuePanic(t *testing.T) {
	err := func() (err error) {
		defer errx.Recover(&err)
		panic("boom")
	}()
	if err == nil {
		t.Fatal("Recover should produce an error")
	}
	if !errx.IsDefect(err) {
		t.Fatal("panic should classify as defect")
	}
	if !errx.Recovered(err) {
		t.Fatal("Recovered should be true")
	}
}

func TestRecoverFromErrorPanic(t *testing.T) {
	sentinel := errors.New("sentinel")
	err := errx.RecoverDo(func() error {
		panic(sentinel)
	})
	if !errors.Is(err, sentinel) {
		t.Fatal("panic(error) should keep the error matchable as cause")
	}
}

func TestRecoverDoNoPanic(t *testing.T) {
	got := errx.RecoverDo(func() error { return nil })
	if got != nil {
		t.Fatalf("RecoverDo should pass through nil, got %v", got)
	}
}

type ctxKey struct{}

func TestContextExtractor(t *testing.T) {
	errx.RegisterContextExtractor(func(ctx context.Context) (string, string, []errx.Field) {
		if v, ok := ctx.Value(ctxKey{}).(string); ok {
			return "trace-" + v, "span-" + v, []errx.Field{{Key: "tenant", Value: v, Safe: true}}
		}
		return "", "", nil
	})
	ctx := context.WithValue(context.Background(), ctxKey{}, "acme")
	e := errx.New("op failed").WithCode("X").Context(ctx)
	if e.TraceID() != "trace-acme" || e.SpanID() != "span-acme" {
		t.Fatalf("ctx not extracted: trace=%q span=%q", e.TraceID(), e.SpanID())
	}
	var sawTenant bool
	for _, f := range e.Fields() {
		if f.Key == "tenant" && f.Value == "acme" {
			sawTenant = true
		}
	}
	if !sawTenant {
		t.Fatal("ctx field not attached")
	}
}

func TestErrtestHelpers(t *testing.T) {
	e := errx.New("x").WithCode("E_X").WithClass(errx.ClassDefect).WithRetryable(0)
	errtest.Code(t, e, "E_X")
	errtest.Class(t, e, errx.ClassDefect)
	errtest.Retryable(t, e, true)
	errtest.HasCode(t, errx.Wrap(e, "ctx"), "E_X")
	errtest.NoError(t, nil)

	// Same construction site, variable message → same fingerprint.
	mk := func(msg string) error {
		return errx.New(msg).WithDomain("d").WithCode("C")
	}
	errtest.Fingerprint(t, mk("user 1 missing"), mk("user 2 missing"))
}
