package graphql_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ubgo/errx"
	graphqlx "github.com/ubgo/errx/contrib/graphql"
)

func TestPresentShapesAndRedacts(t *testing.T) {
	e := errx.New("pg: internal").
		WithDomain("billing").WithCode("NOT_FOUND").
		WithPublic("Order not found").
		WithRetryable(0).
		With("secret", "leak").WithSafe("order_id", "o-3")

	ge := graphqlx.Present(context.Background(), e)
	if ge.Message != "Order not found" {
		t.Fatalf("message = %q, want public message", ge.Message)
	}
	if ge.Extensions["code"] != "NOT_FOUND" || ge.Extensions["domain"] != "billing" {
		t.Fatalf("extensions identity wrong: %+v", ge.Extensions)
	}
	if ge.Extensions["fingerprint"] == nil || ge.Extensions["retryable"] != true {
		t.Fatalf("extensions missing fp/retryable: %+v", ge.Extensions)
	}
	fields, _ := ge.Extensions["fields"].(map[string]any)
	if fields == nil || fields["order_id"] != "o-3" {
		t.Fatalf("safe field missing: %+v", fields)
	}
	if _, leaked := fields["secret"]; leaked {
		t.Fatal("unsafe field leaked into GraphQL extensions")
	}
}

func TestPresentNonErrxFallback(t *testing.T) {
	ge := graphqlx.Present(context.Background(), errors.New("boom"))
	if ge == nil || ge.Message == "" {
		t.Fatalf("fallback produced no error: %+v", ge)
	}
}
