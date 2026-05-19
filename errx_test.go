package errx_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ubgo/errx"
)

func TestNewAndIdentity(t *testing.T) {
	e := errx.New("db: deadlock on user_orders").
		WithDomain("billing").WithCode("TX_RETRY").
		WithClass(errx.ClassExpected).WithSeverity(errx.SevWarn).
		WithPublic("Please retry").WithHint("retry with backoff").
		WithRetryable(2 * time.Second)

	if got := errx.Code(e); got != "TX_RETRY" {
		t.Fatalf("Code = %q, want TX_RETRY", got)
	}
	if e.Domain() != "billing" {
		t.Fatalf("Domain = %q", e.Domain())
	}
	if e.Public("fallback") != "Please retry" {
		t.Fatalf("Public = %q", e.Public("fallback"))
	}
	if !errx.IsRetryable(e) || errx.RetryAfter(e) != 2*time.Second {
		t.Fatalf("retryable=%v after=%v", errx.IsRetryable(e), errx.RetryAfter(e))
	}
	if !errx.IsExpected(e) {
		t.Fatal("want IsExpected")
	}
}

func TestPublicFallback(t *testing.T) {
	e := errx.New("internal detail leak")
	if got := e.Public("Something went wrong"); got != "Something went wrong" {
		t.Fatalf("Public fallback = %q", got)
	}
}

func TestWrapTransparentIsAs(t *testing.T) {
	sentinel := errors.New("pq: deadlock")
	e := errx.Wrap(sentinel, "load orders").WithCode("DB")

	if !errors.Is(e, sentinel) {
		t.Fatal("errors.Is should traverse a transparent Wrap")
	}
	if errx.Code(e) != "DB" {
		t.Fatalf("Code = %q", errx.Code(e))
	}
	if !strings.Contains(e.Error(), "load orders") || !strings.Contains(e.Error(), "pq: deadlock") {
		t.Fatalf("Error() = %q", e.Error())
	}
}

func TestOpaqueBarrier(t *testing.T) {
	sentinel := errors.New("secret dependency sentinel")
	e := errx.Wrap(sentinel, "public boundary").Opaque()
	if errors.Is(e, sentinel) {
		t.Fatal("Opaque() must hide the cause from errors.Is")
	}
}

func TestWrapNilReturnsNil(t *testing.T) {
	if errx.Wrap(nil, "x") != nil {
		t.Fatal("Wrap(nil) must be nil")
	}
	if errx.Wrapf(nil, "x %d", 1) != nil {
		t.Fatal("Wrapf(nil) must be nil")
	}
	if errx.Note(nil, "k", 1) != nil {
		t.Fatal("Note(nil) must be nil")
	}
}

func TestWrapInheritsInnerIdentity(t *testing.T) {
	inner := errx.New("inner").WithCode("INNER").WithClass(errx.ClassDefect)
	outer := errx.Wrap(inner, "outer context")
	if errx.Code(outer) != "INNER" {
		t.Fatalf("inherited Code = %q, want INNER", errx.Code(outer))
	}
	if !errx.IsDefect(outer) {
		t.Fatal("outer should inherit ClassDefect")
	}
}

func TestNoteNoExtraFrame(t *testing.T) {
	e := errx.New("base")
	got := errx.Note(e, "attempt", 3)
	if got != error(e) {
		t.Fatal("Note on *Error should return the same error, no new wrapper")
	}
	fields := e.Fields()
	if len(fields) != 1 || fields[0].Key != "attempt" {
		t.Fatalf("Note field not attached: %+v", fields)
	}
}

func TestJoinAndIs(t *testing.T) {
	a := errors.New("a")
	b := errx.New("b").WithCode("B")
	j := errx.Join(nil, a, nil, b)
	if !errors.Is(j, a) || !errx.HasCode(j, "B") {
		t.Fatalf("Join membership broken: %v", j)
	}
	if errx.Join(nil, nil) != nil {
		t.Fatal("Join of all-nil must be nil")
	}
	if errx.Join(a) != a {
		t.Fatal("Join of single error must return it unchanged")
	}
}

func TestSuppressedMatchableAndKeptSeparate(t *testing.T) {
	primary := errors.New("primary")
	closeErr := errors.New("close failed")
	e := errx.Wrap(primary, "op").Suppress(closeErr)

	if !errors.Is(e, primary) {
		t.Fatal("primary cause must match")
	}
	if !errors.Is(e, closeErr) {
		t.Fatal("suppressed error must remain matchable via errors.Is")
	}
	if len(e.Suppressed()) != 1 {
		t.Fatalf("Suppressed() = %v", e.Suppressed())
	}
}

func TestGenericAs(t *testing.T) {
	target := errx.New("typed").WithCode("X")
	wrapped := errx.Wrap(target, "ctx")
	got, ok := errx.As[*errx.Error](wrapped)
	if !ok || got == nil {
		t.Fatalf("As[*errx.Error] failed: %v %v", got, ok)
	}
}

func TestClassFromContext(t *testing.T) {
	if errx.ClassOf(context.Canceled) != errx.ClassCancelled {
		t.Fatal("context.Canceled should classify as Cancelled")
	}
	if !errx.IsCancelled(fmt.Errorf("wrap: %w", context.DeadlineExceeded)) {
		t.Fatal("wrapped DeadlineExceeded should be Cancelled")
	}
}

func TestFingerprintDeterministicAndMessageIndependent(t *testing.T) {
	mk := func(msg string) string {
		return errx.New(msg).WithDomain("d").WithCode("C").Fingerprint()
	}
	f1 := mk("user 123 not found")
	f2 := mk("user 999 not found")
	if f1 == "" {
		t.Fatal("empty fingerprint")
	}
	if f1 != f2 {
		t.Fatalf("fingerprint must ignore variable message: %s != %s", f1, f2)
	}
	// Different code => different fingerprint.
	other := errx.New("x").WithDomain("d").WithCode("OTHER").Fingerprint()
	if other == f1 {
		t.Fatal("different code must change fingerprint")
	}
}

func TestSnapshotRedactsUnsafeFields(t *testing.T) {
	e := errx.New("op failed").WithDomain("d").WithCode("C").
		With("password", "hunter2").
		WithSafe("user_id", "u-1")

	rep, ok := errx.Snapshot(context.Background(), e)
	if !ok {
		t.Fatal("Snapshot should succeed")
	}
	var sawRedacted, sawSafe bool
	for _, f := range rep.Fields {
		if f.Key == "password" {
			if f.Value != errx.RedactedMarker {
				t.Fatalf("unsafe field not redacted: %v", f.Value)
			}
			sawRedacted = true
		}
		if f.Key == "user_id" && f.Value == "u-1" {
			sawSafe = true
		}
	}
	if !sawRedacted || !sawSafe {
		t.Fatalf("redaction wrong: redacted=%v safe=%v", sawRedacted, sawSafe)
	}
	if rep.Fingerprint == "" || rep.Code != "C" {
		t.Fatalf("report identity wrong: %+v", rep)
	}
}

func TestAccumulate(t *testing.T) {
	err := errx.Accumulate(
		func() error { return nil },
		func() error { return errors.New("bad name") },
		func() error { return errors.New("bad email") },
	)
	if err == nil {
		t.Fatal("Accumulate should return combined error")
	}
	if errx.Code(err) != "VALIDATION" {
		t.Fatalf("Code = %q", errx.Code(err))
	}
	if errx.Accumulate(func() error { return nil }) != nil {
		t.Fatal("all-nil Accumulate must be nil")
	}
}

func TestParAccumulateRace(t *testing.T) {
	fns := make([]func() error, 50)
	for i := range fns {
		i := i
		fns[i] = func() error { return fmt.Errorf("e%d", i) }
	}
	err := errx.ParAccumulate(fns...)
	e := errx.Get(err)
	if e == nil || len(e.Suppressed()) != 50 {
		t.Fatalf("ParAccumulate lost errors: %d", len(e.Suppressed()))
	}
}

func TestRegistryFanOut(t *testing.T) {
	var got *errx.Report
	reg := errx.NewRegistry().Add(sinkFunc(func(_ context.Context, r *errx.Report) error {
		got = r
		return nil
	}))
	reg.Report(context.Background(), errx.New("boom").WithCode("BOOM"))
	if got == nil || got.Code != "BOOM" {
		t.Fatalf("sink did not receive report: %+v", got)
	}
	// No-op for non-errx errors.
	reg.Report(context.Background(), errors.New("plain"))
}

func TestFormatVerbose(t *testing.T) {
	e := errx.New("boom").WithDomain("d").WithCode("C").With("k", "v")
	s := fmt.Sprintf("%+v", e)
	if !strings.Contains(s, "identity: d/C") || !strings.Contains(s, "field:") {
		t.Fatalf("%%+v missing detail:\n%s", s)
	}
	if fmt.Sprintf("%s", e) != "boom" {
		t.Fatalf("%%s = %q", fmt.Sprintf("%s", e))
	}
}

func TestStackCaptured(t *testing.T) {
	e := errx.New("x")
	fr := e.Frames()
	if len(fr) == 0 {
		t.Fatal("expected an origin stack")
	}
	if !strings.Contains(fr[0].Function, "TestStackCaptured") {
		t.Fatalf("top frame should be the caller, got %q", fr[0].Function)
	}
}

type sinkFunc func(context.Context, *errx.Report) error

func (f sinkFunc) Name() string                                   { return "test" }
func (f sinkFunc) Emit(ctx context.Context, r *errx.Report) error { return f(ctx, r) }
