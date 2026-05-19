package errx_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ubgo/errx"
)

func TestEncodeDecodeRoundTripIdentity(t *testing.T) {
	orig := errx.New("internal: pq deadlock on orders").
		WithDomain("billing").WithCode("TX_RETRY").
		WithClass(errx.ClassExpected).WithSeverity(errx.SevWarn).
		WithPublic("Please retry").WithHint("backoff").WithOwner("payments").
		WithRetryable(3*time.Second).
		WithTrace("trace-1", "span-1").
		With("password", "hunter2"). // unsafe — must NOT cross
		WithSafe("order_id", "o-7")

	blob, err := errx.Encode(orig)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if strings.Contains(string(blob), "hunter2") {
		t.Fatal("unsafe field value leaked into the wire form")
	}

	got, err := errx.Decode(blob)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if errx.Code(got) != "TX_RETRY" || got.Domain() != "billing" {
		t.Fatalf("identity lost: code=%q domain=%q", errx.Code(got), got.Domain())
	}
	if !errx.IsRetryable(got) || errx.RetryAfter(got) != 3*time.Second {
		t.Fatalf("retry metadata lost: %v %v", errx.IsRetryable(got), errx.RetryAfter(got))
	}
	if got.Public("x") != "Please retry" || got.SeverityOf() != errx.SevWarn {
		t.Fatalf("message/severity lost: %q %v", got.Public("x"), got.SeverityOf())
	}
	if got.Fingerprint() != orig.Fingerprint() {
		t.Fatalf("fingerprint not preserved across the wire: %s vs %s", got.Fingerprint(), orig.Fingerprint())
	}
	var sawSafe, sawUnsafe bool
	for _, f := range got.Fields() {
		if f.Key == "order_id" {
			sawSafe = true
		}
		if f.Key == "password" {
			sawUnsafe = true
		}
	}
	if !sawSafe || sawUnsafe {
		t.Fatalf("field crossing wrong: safe=%v unsafe=%v", sawSafe, sawUnsafe)
	}
}

func TestDecodedStackSurvives(t *testing.T) {
	blob, _ := errx.Encode(errx.New("boom").WithCode("C"))
	got, _ := errx.Decode(blob)
	if len(got.Frames()) == 0 {
		t.Fatal("stack frames should survive the round trip")
	}
}

func TestMarshalJSONUsable(t *testing.T) {
	b, err := json.Marshal(errx.New("x").WithCode("C"))
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if !strings.Contains(string(b), `"code":"C"`) {
		t.Fatalf("json output unexpected: %s", b)
	}
}

func TestCodeMigrationOnDecode(t *testing.T) {
	errx.RegisterCodeMigration("OLD_QUOTA", "QUOTA_EXCEEDED")
	errx.RegisterCodeMigration("QUOTA_EXCEEDED", "QUOTA_EXCEEDED") // cycle guard no-op

	blob, _ := errx.Encode(errx.New("over limit").WithCode("OLD_QUOTA"))
	got, err := errx.Decode(blob)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if errx.Code(got) != "QUOTA_EXCEEDED" {
		t.Fatalf("retired code not migrated: %q", errx.Code(got))
	}
	// Unmigrated codes pass through unchanged.
	b2, _ := errx.Encode(errx.New("x").WithCode("STILL_FINE"))
	g2, _ := errx.Decode(b2)
	if errx.Code(g2) != "STILL_FINE" {
		t.Fatalf("unmigrated code changed: %q", errx.Code(g2))
	}
}

func TestAccumulatorLenAndCodecEdges(t *testing.T) {
	acc := errx.NewAccumulator()
	if acc.Len() != 0 {
		t.Fatal("empty accumulator Len should be 0")
	}
	acc.Add("a", errors.New("x"))
	acc.AddErr(errors.New("y"))
	acc.Add("nil", nil) // ignored
	if acc.Len() != 2 {
		t.Fatalf("Len = %d, want 2", acc.Len())
	}

	// Encode(nil) → "null"; MarshalJSON of an error with suppressed + cause.
	b, err := errx.Encode(nil)
	if err != nil || string(b) != "null" {
		t.Fatalf("Encode(nil) = %q %v", b, err)
	}
	e := errx.Wrap(errors.New("root"), "ctx").
		WithCode("C").WithRetryable(time.Second).
		Suppress(errors.New("sup"))
	raw, err := e.MarshalJSON()
	if err != nil || !strings.Contains(string(raw), `"code":"C"`) ||
		!strings.Contains(string(raw), `"suppressed"`) {
		t.Fatalf("MarshalJSON missing fields: %s (%v)", raw, err)
	}
	got, err := errx.Decode(raw)
	if err != nil || errx.Code(got) != "C" || !errx.IsRetryable(got) {
		t.Fatalf("decode round trip: code=%q retry=%v err=%v", errx.Code(got), errx.IsRetryable(got), err)
	}
}

func TestEncodeNonErrx(t *testing.T) {
	b, err := errx.Encode(plainErr("just text"))
	if err != nil {
		t.Fatalf("Encode plain: %v", err)
	}
	got, err := errx.Decode(b)
	if err != nil || got.Error() != "just text" {
		t.Fatalf("plain envelope round trip: %v %q", err, got.Error())
	}
}

type plainErr string

func (p plainErr) Error() string { return string(p) }
