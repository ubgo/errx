package diag_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/ubgo/errx"
	"github.com/ubgo/errx/diag"
)

func TestGraphicalWithSourceSpan(t *testing.T) {
	src := "SELECT id, name\nFROM   users\nWHERE  emial = $1"
	bad := strings.Index(src, "emial")
	e := errx.New("unknown column \"emial\"").
		WithCode("PG_UNDEFINED_COLUMN").
		WithSeverity(errx.SevError).
		WithHint("did you mean \"email\"?").
		WithURL("https://errors.example.com/PG_UNDEFINED_COLUMN").
		WithSource("query.sql", src).
		WithLabel(bad, len("emial"), "no such column")

	out := diag.String(e) // zero Options = deterministic, no color

	mustContain(t, out, "error[PG_UNDEFINED_COLUMN]: unknown column \"emial\"")
	mustContain(t, out, "--> query.sql")
	mustContain(t, out, "3 | WHERE  emial = $1")
	mustContain(t, out, "^^^^^ no such column")
	mustContain(t, out, "help: did you mean \"email\"?")
	mustContain(t, out, "docs: https://errors.example.com/PG_UNDEFINED_COLUMN")
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("zero Options must be color-free:\n%s", out)
	}
	// Caret must sit under the bad token (7 spaces: "WHERE  ").
	if !strings.Contains(out, "| "+strings.Repeat(" ", 7)+"^^^^^") {
		t.Fatalf("caret misaligned:\n%s", out)
	}
}

func TestNarratable(t *testing.T) {
	src := "a = 1\nb = oops"
	e := errx.New("invalid value").WithCode("CFG").
		WithHint("expected an integer").
		WithSource("app.conf", src).
		WithLabel(strings.Index(src, "oops"), 4, "not a number")

	out := diag.String(e, diag.Options{Narratable: true})
	mustContain(t, out, "Error [CFG]: invalid value")
	mustContain(t, out, "at app.conf line 2, column 5: not a number")
	mustContain(t, out, "help: expected an integer")
	if strings.Contains(out, "^") {
		t.Fatalf("narratable form should not draw carets:\n%s", out)
	}
}

func TestColorOption(t *testing.T) {
	e := errx.New("boom").WithCode("X")
	if !strings.Contains(diag.String(e, diag.Options{Color: true}), "\x1b[") {
		t.Fatal("Color:true should emit ANSI")
	}
}

func TestNonErrxAndNoSource(t *testing.T) {
	if got := diag.String(errors.New("plain")); strings.TrimSpace(got) != "plain" {
		t.Fatalf("non-errx render = %q", got)
	}
	out := diag.String(errx.New("no source").WithCode("C"))
	mustContain(t, out, "error[C]: no source")
	if strings.Contains(out, "-->") {
		t.Fatal("should not render a source block when none is set")
	}
}

func mustContain(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Fatalf("missing %q in:\n%s", sub, s)
	}
}
