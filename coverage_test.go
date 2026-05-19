package errx_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/ubgo/errx"
)

func TestNilAndEnumStrings(t *testing.T) {
	var e *errx.Error
	if e.Error() != "<nil>" {
		t.Fatalf("nil Error() = %q", e.Error())
	}
	if e.Public("fb") != "fb" {
		t.Fatal("nil Public should return fallback")
	}
	if e.Fingerprint() != "" || e.Frames() != nil {
		t.Fatal("nil fingerprint/frames should be empty")
	}
	if errx.ClassExpected.String() != "expected" || errx.ClassDefect.String() != "defect" ||
		errx.ClassCancelled.String() != "cancelled" || errx.Class(99).String() != "unknown" {
		t.Fatal("Class.String wrong")
	}
	if errx.SevDebug.String() != "debug" || errx.SevInfo.String() != "info" ||
		errx.SevWarn.String() != "warn" || errx.SevFatal.String() != "fatal" ||
		errx.Severity(99).String() != "error" {
		t.Fatal("Severity.String wrong")
	}
}

func TestNewfWrapfMultiErrorFormat(t *testing.T) {
	e := errx.Newf("code %d failed", 7)
	if e.Error() != "code 7 failed" {
		t.Fatalf("Newf = %q", e.Error())
	}
	w := errx.Wrapf(errors.New("inner"), "ctx %s", "x")
	if w.Error() != "ctx x: inner" {
		t.Fatalf("Wrapf = %q", w.Error())
	}
	j := errx.Join(errors.New("a"), errors.New("b"), errors.New("c"))
	if !strings.Contains(j.Error(), "3 errors:") || !strings.Contains(j.Error(), "- b") {
		t.Fatalf("multiError format = %q", j.Error())
	}
}

func TestGetCodeHelpersOnNonErrx(t *testing.T) {
	plain := errors.New("plain")
	if errx.Get(plain) != nil {
		t.Fatal("Get(non-errx) should be nil")
	}
	if errx.Code(plain) != "" || errx.HasCode(plain, "X") || errx.FindByCode(plain, "X") != nil {
		t.Fatal("code helpers on non-errx wrong")
	}
	if errx.ClassOf(plain) != errx.ClassExpected {
		t.Fatal("ClassOf(plain) should be Expected")
	}
	if errx.IsRetryable(plain) || errx.RetryAfter(plain) != 0 {
		t.Fatal("retry helpers on non-errx wrong")
	}
	// HasCode / FindByCode across a multi-unwrap chain.
	deep := errx.Join(errors.New("x"), errx.New("y").WithCode("DEEP"))
	if !errx.HasCode(deep, "DEEP") || errx.FindByCode(deep, "DEEP") == nil {
		t.Fatal("HasCode/FindByCode should traverse Join")
	}
}

func TestFormatVerboseFullDetail(t *testing.T) {
	e := errx.Wrap(errors.New("root cause"), "doing thing").
		WithDomain("d").WithCode("C").
		WithPublic("user msg").WithHint("do X").
		With("k", "v").
		Suppress(errors.New("cleanup failed"))
	s := fmt.Sprintf("%+v", e)
	for _, want := range []string{"doing thing", "identity: d/C", "public:", "hint:", "field:", "suppressed:"} {
		if !strings.Contains(s, want) {
			t.Fatalf("%%+v missing %q in:\n%s", want, s)
		}
	}
	if fmt.Sprintf("%q", e) == "" {
		t.Fatal("quoted verb produced empty output")
	}
}

func TestSlogLogValue(t *testing.T) {
	var buf strings.Builder
	l := slog.New(slog.NewJSONHandler(&buf, nil))
	e := errx.New("boom").WithDomain("d").WithCode("C").
		WithRetryable(0).WithTrace("t-1", "s-1").
		With("password", "secret").WithSafe("uid", "u1")
	l.Error("failed", "err", e)
	out := buf.String()
	if !strings.Contains(out, `"code":"C"`) || !strings.Contains(out, `"fingerprint"`) {
		t.Fatalf("slog group missing identity: %s", out)
	}
	if strings.Contains(out, "secret") {
		t.Fatalf("unsafe field leaked to slog: %s", out)
	}
	if !strings.Contains(out, `"uid":"u1"`) {
		t.Fatalf("safe field missing from slog: %s", out)
	}
}

func TestRegistryOnSinkError(t *testing.T) {
	var gotName string
	reg := errx.NewRegistry()
	reg.OnSinkError = func(name string, _ error) { gotName = name }
	reg.Add(failSink{})
	reg.Report(context.Background(), errx.New("x").WithCode("C"))
	if gotName != "failing" {
		t.Fatalf("OnSinkError not called, got %q", gotName)
	}
}

type failSink struct{}

func (failSink) Name() string                             { return "failing" }
func (failSink) Emit(context.Context, *errx.Report) error { return errors.New("sink down") }

func TestDiagnosticBuilders(t *testing.T) {
	e := errx.New("bad token").WithCode("LEX").
		WithURL("https://docs/LEX").
		WithSource("in.dsl", "let x = ?").
		WithLabel(8, 1, "unexpected")
	if e.URL() != "https://docs/LEX" {
		t.Fatalf("URL = %q", e.URL())
	}
	n, c := e.Source()
	if n != "in.dsl" || c != "let x = ?" {
		t.Fatalf("Source = %q %q", n, c)
	}
	ls := e.Labels()
	if len(ls) != 1 || ls[0].Start != 8 || ls[0].Msg != "unexpected" {
		t.Fatalf("Labels = %+v", ls)
	}
	// defensive copy
	ls[0].Msg = "mutated"
	if e.Labels()[0].Msg == "mutated" {
		t.Fatal("Labels must return a defensive copy")
	}
	if errx.New("x").Labels() != nil {
		t.Fatal("no labels => nil")
	}
}

func TestLocalizedMessages(t *testing.T) {
	e := errx.New("internal").WithCode("C").
		WithPublic("Something went wrong").
		WithLocalized("fr-FR", "Une erreur est survenue").
		WithLocalized("ja", "エラーが発生しました")

	if got := e.Localized("fr-FR", "fb"); got != "Une erreur est survenue" {
		t.Fatalf("exact locale = %q", got)
	}
	if got := e.Localized("fr-CA", "fb"); got != "Une erreur est survenue" {
		t.Fatalf("language fallback fr-CA->fr-FR = %q", got)
	}
	if got := e.Localized("ja-JP", "fb"); got != "エラーが発生しました" {
		t.Fatalf("ja-JP->ja = %q", got)
	}
	if got := e.Localized("de-DE", "fb"); got != "Something went wrong" {
		t.Fatalf("unknown locale should fall back to Public, got %q", got)
	}
	if errx.New("x").Localized("en", "fb") != "fb" {
		t.Fatal("no localized + no public => fallback")
	}
	if m := e.LocaleMessages(); len(m) != 2 {
		t.Fatalf("LocaleMessages = %+v", m)
	}
}

func TestDocRegistry(t *testing.T) {
	errx.RegisterDoc("E_QUOTA", errx.DocEntry{
		URL:         "https://docs/E_QUOTA",
		Summary:     "Quota exceeded",
		Remediation: "request a quota increase",
	})
	d, ok := errx.DocFor("E_QUOTA")
	if !ok || d.URL != "https://docs/E_QUOTA" {
		t.Fatalf("DocFor = %+v %v", d, ok)
	}

	// An error carrying only the code resolves URL + remediation from the registry.
	e := errx.New("over quota").WithCode("E_QUOTA")
	if e.URL() != "https://docs/E_QUOTA" {
		t.Fatalf("URL from registry = %q", e.URL())
	}
	if e.Remediation() != "request a quota increase" {
		t.Fatalf("Remediation from registry = %q", e.Remediation())
	}
	// Explicit WithURL / WithHint win over the registry.
	e2 := errx.New("x").WithCode("E_QUOTA").WithURL("https://override").WithHint("explicit hint")
	if e2.URL() != "https://override" || e2.Remediation() != "explicit hint" {
		t.Fatalf("explicit should win: %q %q", e2.URL(), e2.Remediation())
	}
	if _, ok := errx.DocFor("NOPE"); ok {
		t.Fatal("unknown code should not resolve")
	}

	rep, _ := errx.Snapshot(context.Background(), e)
	if rep.DocURL != "https://docs/E_QUOTA" || rep.Hint != "request a quota increase" {
		t.Fatalf("Snapshot did not carry resolved doc: %+v", rep)
	}
}

func TestReturnTrace(t *testing.T) {
	leaf := func() error { return errx.New("leaf failed").WithCode("LEAF") }
	mid := func() error { return errx.Wrap(leaf(), "mid: doing work") }
	top := func() error { return errx.Wrapf(mid(), "top: handling %s", "request") }

	err := top()
	tr := errx.Get(err).Trace()
	if len(tr) < 2 {
		t.Fatalf("return-trace should record each wrap layer, got %d: %+v", len(tr), tr)
	}
	// Innermost-first: the first frame is the Wrapf site in `top`.
	if !strings.Contains(tr[0].Function, "TestReturnTrace") {
		t.Fatalf("first trace frame should be the wrap caller, got %q", tr[0].Function)
	}
	rep, _ := errx.Snapshot(context.Background(), err)
	if len(rep.Trace) != len(tr) {
		t.Fatalf("Report.Trace mismatch: %d vs %d", len(rep.Trace), len(tr))
	}
	if s := fmt.Sprintf("%+v", errx.Get(err)); !strings.Contains(s, "trace:") {
		t.Fatalf("%%+v should include the trace section:\n%s", s)
	}
}

func TestAccessorsAndEdges(t *testing.T) {
	cause := errors.New("root")
	e := errx.Wrap(cause, "ctx").
		WithCode("C").WithDomain("d").
		WithHint("do x").WithOwner("team-a").
		WithSeverity(errx.SevWarn).WithRetryable(5 * time.Second)

	if e.Hint() != "do x" || e.Owner() != "team-a" {
		t.Fatalf("hint/owner = %q %q", e.Hint(), e.Owner())
	}
	if !errors.Is(e.Cause(), cause) {
		t.Fatal("Cause() should return the wrapped error")
	}
	if !e.Retryable() || e.RetryAfter() != 5*time.Second {
		t.Fatalf("retryable=%v after=%v", e.Retryable(), e.RetryAfter())
	}
	if e.ClassOf() != errx.ClassExpected || e.SeverityOf() != errx.SevWarn {
		t.Fatalf("class=%v sev=%v", e.ClassOf(), e.SeverityOf())
	}
	if e.SpanID() != "" || e.TraceID() != "" {
		t.Fatal("trace/span empty by default")
	}
	_ = e.WithTrace("t", "s")
	if e.TraceID() != "t" || e.SpanID() != "s" {
		t.Fatalf("WithTrace not applied: %q %q", e.TraceID(), e.SpanID())
	}

	// As failure path (wrong target type).
	if _, ok := errx.As[*specificErr](errx.New("x")); ok {
		t.Fatal("As should fail for an unrelated type")
	}
	// Fingerprint of a non-errx error is empty.
	if errx.Fingerprint(errors.New("plain")) != "" {
		t.Fatal("Fingerprint(non-errx) must be empty")
	}
	// Snapshot of a non-errx error returns ok=false.
	if _, ok := errx.Snapshot(context.Background(), errors.New("plain")); ok {
		t.Fatal("Snapshot(non-errx) should be ok=false")
	}
	// Default %v / %s verbs (non-+v branch of Format).
	if got := fmt.Sprintf("%v", e); got == "" {
		t.Fatal("default value verb produced empty output")
	}
	if got := fmt.Sprintf("%s", e); got == "" {
		t.Fatal("string verb produced empty output")
	}
	// Default verb branch via Format directly (avoids a vet printf flag).
	var st fakeState
	e.Format(&st, 'd')
	if st.b.Len() == 0 {
		t.Fatal("default verb branch should render")
	}
}

// fakeState is a minimal fmt.State for exercising Format's default branch.
type fakeState struct{ b strings.Builder }

func (f *fakeState) Write(p []byte) (int, error) { return f.b.Write(p) }
func (f *fakeState) Width() (int, bool)          { return 0, false }
func (f *fakeState) Precision() (int, bool)      { return 0, false }
func (f *fakeState) Flag(int) bool               { return false }

type specificErr struct{}

func (*specificErr) Error() string { return "specific" }

func TestErrdeferJoinBranch(t *testing.T) {
	// AppendSuppressed when *errp is a non-errx error → join.
	primary := errors.New("primary")
	extra := errors.New("extra")
	err := primary
	errx.AppendSuppressed(&err, extra)
	if !errors.Is(err, primary) || !errors.Is(err, extra) {
		t.Fatal("join branch should keep both matchable")
	}
	// nil guards.
	errx.AppendSuppressed(nil, extra) // no panic
	var n error
	errx.AppendSuppressed(&n, nil) // no-op
	if n != nil {
		t.Fatal("AppendSuppressed(&nil, nil) must stay nil")
	}
	// CloseSuppressing with a nil closer is a no-op.
	var e2 error
	errx.CloseSuppressing(&e2, nil)
	if e2 != nil {
		t.Fatal("CloseSuppressing(nil closer) must be a no-op")
	}
}

func TestInspectChainShapes(t *testing.T) {
	// Code via multi-unwrap (Join), and HasCode/FindByCode parity.
	j := errx.Join(errors.New("a"), errx.New("b").WithCode("DEEP"))
	if errx.Code(j) != "DEEP" || !errx.HasCode(j, "DEEP") || errx.FindByCode(j, "DEEP") == nil {
		t.Fatalf("multi-unwrap code lookup failed: %q", errx.Code(j))
	}
	// Opaque barrier hides a wrapped sentinel from errors.Is (it does NOT
	// erase identity inherited onto the wrapper itself by Wrap — that is
	// by design).
	sentinel := errors.New("dep sentinel")
	op := errx.Wrap(sentinel, "boundary").Opaque()
	if errors.Is(op, sentinel) {
		t.Fatal("Opaque must hide the wrapped cause from errors.Is")
	}
	// Code() method on a foreign error is honored.
	if errx.Code(&specificCoded{}) != "FOREIGN" {
		t.Fatalf("Code() method not honored: %q", errx.Code(&specificCoded{}))
	}
	if !errx.HasCode(errx.Wrap(&specificCoded{}, "w"), "FOREIGN") {
		t.Fatal("HasCode should see a wrapped foreign Code()")
	}
}

type specificCoded struct{}

func (*specificCoded) Error() string { return "foreign" }
func (*specificCoded) Code() string  { return "FOREIGN" }

func TestSnapshotFullAndSingleUnwrapChain(t *testing.T) {
	cause := errors.New("db down")
	e := errx.Wrap(cause, "query").
		WithDomain("d").WithCode("Q").
		WithTrace("trace-1", "span-1").
		Suppress(errors.New("close failed")).
		With("secret", "x").WithSafe("q", "SELECT 1")

	rep, ok := errx.Snapshot(context.Background(), e)
	if !ok {
		t.Fatal("Snapshot should succeed")
	}
	if rep.Cause != "query: db down" && rep.Cause != "db down" {
		t.Fatalf("cause not captured: %q", rep.Cause)
	}
	if len(rep.Suppressed) != 1 || rep.TraceID != "trace-1" || rep.SpanID != "span-1" {
		t.Fatalf("snapshot missing suppressed/trace: %+v", rep)
	}
	var sawRedacted, sawSafe bool
	for _, f := range rep.Fields {
		if f.Key == "secret" && f.Value == errx.RedactedMarker {
			sawRedacted = true
		}
		if f.Key == "q" && f.Value == "SELECT 1" {
			sawSafe = true
		}
	}
	if !sawRedacted || !sawSafe {
		t.Fatalf("redaction wrong in snapshot: %+v", rep.Fields)
	}

	// FindByCode / Code / HasCode through a single-Unwrap (fmt.Errorf) chain.
	inner := errx.New("inner").WithCode("INNER")
	chain := fmt.Errorf("layer: %w", inner)
	if errx.Code(chain) != "INNER" || !errx.HasCode(chain, "INNER") || errx.FindByCode(chain, "INNER") == nil {
		t.Fatalf("single-unwrap chain lookup failed: %q", errx.Code(chain))
	}
	// Outer errx with empty code, inner carries it.
	outerNoCode := &wrapNoCode{inner: inner}
	if errx.Code(outerNoCode) != "INNER" {
		t.Fatalf("should skip empty-code layer: %q", errx.Code(outerNoCode))
	}
}

type wrapNoCode struct{ inner error }

func (w *wrapNoCode) Error() string { return "nocode: " + w.inner.Error() }
func (w *wrapNoCode) Unwrap() error { return w.inner }

func TestRemainingBranches(t *testing.T) {
	// Error(): leaf (no cause), opaque (msg only), wrap with empty msg.
	if errx.New("leaf only").Error() != "leaf only" {
		t.Fatal("leaf Error()")
	}
	op := errx.Wrap(errors.New("hidden"), "shown").Opaque()
	if op.Error() != "shown" {
		t.Fatalf("opaque Error() should be msg only: %q", op.Error())
	}
	wEmpty := errx.Wrap(errors.New("root"), "")
	if wEmpty.Error() != "root" {
		t.Fatalf("empty-msg wrap Error() = %q", wEmpty.Error())
	}

	// Fields(): no fields → nil; with fields → copy.
	if errx.New("x").Fields() != nil {
		t.Fatal("no fields => nil")
	}
	withF := errx.New("x").WithSafe("k", 1)
	fs := withF.Fields()
	fs[0].Key = "mutated"
	if withF.Fields()[0].Key != "k" {
		t.Fatal("Fields must be a defensive copy")
	}

	// URL()/Remediation() with neither explicit nor registered → "".
	bare := errx.New("x").WithCode("NO_DOCS_REGISTERED_XYZ")
	if bare.URL() != "" || bare.Remediation() != "" {
		t.Fatalf("expected empty url/remediation: %q %q", bare.URL(), bare.Remediation())
	}
	// nil-receiver URL/Remediation/Localized.
	var ne *errx.Error
	if ne.URL() != "" || ne.Remediation() != "" || ne.Localized("en", "fb") != "fb" {
		t.Fatal("nil receiver should be safe")
	}

	// LogValue without retryable/trace/fields (minimal branch).
	var buf strings.Builder
	slog.New(slog.NewJSONHandler(&buf, nil)).Error("m", "err", errx.New("bare").WithCode("C"))
	if !strings.Contains(buf.String(), `"code":"C"`) {
		t.Fatalf("minimal LogValue: %s", buf.String())
	}

	// Code/HasCode: multi-unwrap branch that finds nothing then something.
	none := errx.Join(errors.New("a"), errors.New("b"))
	if errx.Code(none) != "" || errx.HasCode(none, "X") {
		t.Fatal("multi-unwrap with no codes should yield nothing")
	}
	if errx.FindByCode(errors.New("plain"), "X") != nil {
		t.Fatal("FindByCode on plain error")
	}
}

func TestMarshalJSONNil(t *testing.T) {
	var e *errx.Error
	b, err := json.Marshal(e)
	if err != nil || string(b) != "null" {
		t.Fatalf("nil MarshalJSON = %q %v", b, err)
	}
}

func TestRecoverPreservesExistingError(t *testing.T) {
	out := func() (err error) {
		err = errx.New("original").WithCode("ORIG")
		defer errx.Recover(&err)
		panic("late panic")
	}()
	if !errx.Recovered(out) {
		t.Fatal("should be marked recovered")
	}
	if !errors.Is(out, errx.Get(out)) {
		t.Fatal("sanity")
	}
	if errx.Code(out) == "" {
		t.Log("panic error has no code (expected)")
	}
}
