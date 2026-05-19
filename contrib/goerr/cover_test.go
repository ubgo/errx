package goerr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ubgo/errx"
	shim "github.com/ubgo/errx/contrib/goerr"
	gerr "github.com/ubgo/goerr"
)

func TestSeverityLevelRoundTripAllCases(t *testing.T) {
	levels := []struct {
		level string
		sev   errx.Severity
	}{
		{"debug", errx.SevDebug},
		{"info", errx.SevInfo},
		{"warn", errx.SevWarn},
		{"warning", errx.SevWarn},
		{"error", errx.SevError},
		{"fatal", errx.SevFatal},
		{"critical", errx.SevFatal},
		{"", errx.SevError},      // unknown → error
		{"bogus", errx.SevError}, // unknown → error
	}
	for _, c := range levels {
		g := gerr.New("m", gerr.WithLevel(c.level))
		if got := shim.From(g).SeverityOf(); got != c.sev {
			t.Fatalf("level %q → severity %v, want %v", c.level, got, c.sev)
		}
	}

	// errx severity → goerr level (To direction), all severities.
	sevs := []struct {
		sev   errx.Severity
		level string
	}{
		{errx.SevDebug, "debug"},
		{errx.SevInfo, "info"},
		{errx.SevWarn, "warn"},
		{errx.SevError, "error"},
		{errx.SevFatal, "fatal"},
	}
	for _, c := range sevs {
		e := errx.New("m").WithCode("C").WithSeverity(c.sev)
		if got := shim.To(e).Response().Level; got != c.level {
			t.Fatalf("severity %v → level %q, want %q", c.sev, got, c.level)
		}
	}
}

func TestFromNonGoerrWraps(t *testing.T) {
	plain := errors.New("plain failure")
	e := shim.From(plain)
	if e == nil || !errors.Is(e, plain) {
		t.Fatal("From(non-goerr) should wrap and stay matchable")
	}
}

func TestFromCarriesDataField(t *testing.T) {
	g := gerr.New("m", gerr.WithCode("C"), gerr.WithData(map[string]int{"n": 1}))
	e := shim.From(g)
	var sawData bool
	for _, f := range e.Fields() {
		if f.Key == "data" {
			sawData = true
		}
	}
	if !sawData {
		t.Fatalf("From should carry Data as a field: %+v", e.Fields())
	}
}

func TestToNonErrx(t *testing.T) {
	g := shim.To(errors.New("just text"))
	if g == nil || g.Error() != "just text" {
		t.Fatalf("To(non-errx) = %v", g)
	}
}

func TestFromWrappedChain(t *testing.T) {
	g := gerr.New("inner", gerr.WithCode("INNER"))
	if errx.Code(shim.From(fmt.Errorf("outer: %w", g))) != "INNER" {
		t.Fatal("From should locate goerr through a wrapped chain")
	}
}
