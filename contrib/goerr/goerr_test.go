package goerr_test

import (
	"fmt"
	"testing"

	"github.com/ubgo/errx"
	shim "github.com/ubgo/errx/contrib/goerr"
	gerr "github.com/ubgo/goerr"
)

func TestFromGoerrPreservesIdentity(t *testing.T) {
	g := gerr.New("db deadlock",
		gerr.WithCode("TX_RETRY"),
		gerr.WithMessageUser("Please retry"),
		gerr.WithTraceID("trace-9"),
		gerr.WithLevel("warn"),
		gerr.WithKV("order_id", "o-1"),
	)
	e := shim.From(g)
	if errx.Code(e) != "TX_RETRY" {
		t.Fatalf("code = %q", errx.Code(e))
	}
	if e.Public("x") != "Please retry" {
		t.Fatalf("public = %q", e.Public("x"))
	}
	if e.TraceID() != "trace-9" || e.SeverityOf() != errx.SevWarn {
		t.Fatalf("trace/severity lost: %q %v", e.TraceID(), e.SeverityOf())
	}
	var sawKV bool
	for _, f := range e.Fields() {
		if f.Key == "order_id" && f.Value == "o-1" {
			sawKV = true
		}
	}
	if !sawKV {
		t.Fatal("KV not carried into errx fields")
	}
}

func TestFromWrappedGoerr(t *testing.T) {
	g := gerr.New("inner", gerr.WithCode("INNER"))
	wrapped := fmt.Errorf("outer: %w", g)
	if errx.Code(shim.From(wrapped)) != "INNER" {
		t.Fatal("From should find goerr inside a wrapped chain")
	}
}

func TestToGoerrRoundTripsAndRedacts(t *testing.T) {
	e := errx.New("internal").
		WithCode("NOT_FOUND").
		WithPublic("Not found").
		WithSeverity(errx.SevWarn).
		With("secret", "leak").
		WithSafe("order_id", "o-2")

	g := shim.To(e)
	r := g.Response()
	if r.Code != "NOT_FOUND" || r.MessageUser != "Not found" || r.Level != "warn" {
		t.Fatalf("goerr payload wrong: %+v", r)
	}
	if _, leaked := r.KV["secret"]; leaked {
		t.Fatal("unsafe field leaked through the bridge")
	}
	if r.KV["order_id"] != "o-2" {
		t.Fatalf("safe field missing: %+v", r.KV)
	}
}

func TestNilPassthrough(t *testing.T) {
	if shim.From(nil) != nil || shim.To(nil) != nil {
		t.Fatal("nil must pass through as nil")
	}
}
