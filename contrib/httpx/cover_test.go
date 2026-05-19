package httpx_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ubgo/errx"
	"github.com/ubgo/errx/contrib/httpx"
)

func TestRegisterStatusAndType(t *testing.T) {
	httpx.RegisterStatus("PAYMENT_REQUIRED_X", http.StatusPaymentRequired)
	httpx.RegisterType("PAYMENT_REQUIRED_X", "https://errors.example.com/pay")
	e := errx.New("nope").WithCode("PAYMENT_REQUIRED_X").WithPublic("Pay up")

	p, ok := httpx.FromError(context.Background(), e)
	if !ok || p.Status != http.StatusPaymentRequired {
		t.Fatalf("RegisterStatus not applied: %+v", p)
	}
	if p.Type != "https://errors.example.com/pay" {
		t.Fatalf("RegisterType not applied: %q", p.Type)
	}
}

func TestProblemTypeFallbacks(t *testing.T) {
	// No code → about:blank.
	p1, _ := httpx.FromError(context.Background(), errx.New("x"))
	if p1.Type != "about:blank" {
		t.Fatalf("no code → type %q, want about:blank", p1.Type)
	}

	// Code present, TypeBaseURL set, no explicit RegisterType → prefix+code.
	old := httpx.TypeBaseURL
	httpx.TypeBaseURL = "https://err.example.com/"
	defer func() { httpx.TypeBaseURL = old }()
	p2, _ := httpx.FromError(context.Background(), errx.New("x").WithCode("UNMAPPED_ABC"))
	if p2.Type != "https://err.example.com/UNMAPPED_ABC" {
		t.Fatalf("TypeBaseURL prefix not applied: %q", p2.Type)
	}

	// Code present but TypeBaseURL == about:blank → about:blank.
	httpx.TypeBaseURL = "about:blank"
	p3, _ := httpx.FromError(context.Background(), errx.New("x").WithCode("STILL_BLANK"))
	if p3.Type != "about:blank" {
		t.Fatalf("about:blank base → type %q", p3.Type)
	}
}

func TestFromErrorNonErrxAndWriteFallback(t *testing.T) {
	if _, ok := httpx.FromError(context.Background(), errors.New("plain")); ok {
		t.Fatal("FromError(non-errx) should be ok=false")
	}
	rec := httptest.NewRecorder()
	// nil request exercises the r==nil branches of Write.
	status := httpx.Write(rec, nil, errors.New("plain"))
	if status != http.StatusInternalServerError {
		t.Fatalf("Write(non-errx) status = %d, want 500", status)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type = %q", ct)
	}
}

func TestStatusForRetryableAndCancelled(t *testing.T) {
	if httpx.StatusFor(errx.New("x").WithRetryable(0)) != http.StatusServiceUnavailable {
		t.Fatal("retryable expected → 503")
	}
	if httpx.StatusFor(errx.New("x").WithClass(errx.ClassCancelled)) != 499 {
		t.Fatal("cancelled → 499")
	}
	if httpx.StatusFor(errx.New("x").WithClass(errx.ClassDefect)) != http.StatusInternalServerError {
		t.Fatal("defect → 500")
	}
	if httpx.StatusFor(errors.New("plain")) != http.StatusInternalServerError {
		t.Fatal("non-errx → 500")
	}
}
