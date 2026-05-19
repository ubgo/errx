package graphql_test

import (
	"context"
	"testing"
	"time"

	"github.com/ubgo/errx"
	graphqlx "github.com/ubgo/errx/contrib/graphql"
)

func TestPresentAllExtensionBranches(t *testing.T) {
	e := errx.New("internal").
		WithDomain("billing").WithCode("RATE").
		WithPublic("Slow down").
		WithRetryable(2*time.Second).
		WithTrace("trace-7", "span-7").
		WithURL("https://docs/RATE").
		WithLocalized("fr-FR", "Ralentissez").
		WithSafe("tenant", "acme")

	ge := graphqlx.Present(context.Background(), e)
	ext := ge.Extensions
	for _, k := range []string{"code", "domain", "class", "fingerprint", "retryable", "retryAfter", "traceId", "docUrl", "localized", "fields"} {
		if ext[k] == nil {
			t.Fatalf("extension %q missing: %+v", k, ext)
		}
	}
	if ge.Message != "Slow down" {
		t.Fatalf("message = %q", ge.Message)
	}
}

func TestPresentNoPublicFallback(t *testing.T) {
	ge := graphqlx.Present(context.Background(), errx.New("x").WithCode("C"))
	if ge.Message != "Internal server error" {
		t.Fatalf("no-public fallback = %q", ge.Message)
	}
}
