package httpx_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ubgo/errx"
	"github.com/ubgo/errx/contrib/httpx"
)

func TestStatusForCodeMapping(t *testing.T) {
	cases := map[string]int{
		"NOT_FOUND":         http.StatusNotFound,
		"VALIDATION":        http.StatusUnprocessableEntity,
		"UNAUTHENTICATED":   http.StatusUnauthorized,
		"PERMISSION_DENIED": http.StatusForbidden,
	}
	for code, want := range cases {
		e := errx.New("x").WithCode(code)
		if got := httpx.StatusFor(e); got != want {
			t.Fatalf("StatusFor(%s) = %d, want %d", code, got, want)
		}
	}
}

func TestStatusForClassFallback(t *testing.T) {
	if got := httpx.StatusFor(errx.New("bug").WithClass(errx.ClassDefect)); got != 500 {
		t.Fatalf("defect = %d, want 500", got)
	}
	if got := httpx.StatusFor(errx.New("gone").WithClass(errx.ClassCancelled)); got != 499 {
		t.Fatalf("cancelled = %d, want 499", got)
	}
	if got := httpx.StatusFor(errx.New("retry").WithRetryable(0)); got != http.StatusServiceUnavailable {
		t.Fatalf("retryable expected = %d, want 503", got)
	}
	if got := httpx.StatusFor(errors.New("plain")); got != 500 {
		t.Fatalf("non-errx = %d, want 500", got)
	}
}

func TestFromErrorRedactsAndShapesRFC9457(t *testing.T) {
	e := errx.New("db exploded internally").
		WithCode("NOT_FOUND").
		WithPublic("Order not found").
		With("secret", "leak-me").
		WithSafe("order_id", "o-1")

	p, ok := httpx.FromError(context.Background(), e)
	if !ok {
		t.Fatal("FromError should succeed")
	}
	if p.Status != http.StatusNotFound || p.Code != "NOT_FOUND" {
		t.Fatalf("bad problem: %+v", p)
	}
	if p.Detail != "Order not found" {
		t.Fatalf("detail should be the public message, got %q", p.Detail)
	}
	if _, leaked := p.Fields["secret"]; leaked {
		t.Fatal("unsafe field leaked into problem+json")
	}
	if p.Fields["order_id"] != "o-1" {
		t.Fatalf("safe field missing: %+v", p.Fields)
	}
}

func TestProblemCarriesLocalizedAndDocURL(t *testing.T) {
	errx.RegisterDoc("RATE", errx.DocEntry{URL: "https://docs/RATE"})
	e := errx.New("internal").WithCode("RATE").
		WithPublic("Slow down").
		WithLocalized("fr-FR", "Ralentissez")

	p, _ := httpx.FromError(context.Background(), e)
	if p.DocURL != "https://docs/RATE" {
		t.Fatalf("docUrl from registry not propagated: %q", p.DocURL)
	}
	if p.Localized["fr-FR"] != "Ralentissez" {
		t.Fatalf("localized not propagated: %+v", p.Localized)
	}
}

func TestWriteResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/orders/9", nil)
	e := errx.New("internal").WithCode("NOT_FOUND").WithPublic("Order not found")

	status := httpx.Write(rec, req, e)
	if status != http.StatusNotFound {
		t.Fatalf("status = %d", status)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type = %q", ct)
	}
	var p httpx.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("body not valid problem+json: %v", err)
	}
	if p.Instance != "/orders/9" || p.Detail != "Order not found" {
		t.Fatalf("problem body wrong: %+v", p)
	}
}
