package connect_test

import (
	"context"
	"testing"

	connectrpc "connectrpc.com/connect"
	"github.com/ubgo/errx"
	errxconnect "github.com/ubgo/errx/contrib/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

func TestCodeMapping(t *testing.T) {
	if errxconnect.CodeFor(errx.New("x").WithCode("NOT_FOUND")) != connectrpc.CodeNotFound {
		t.Fatal("NOT_FOUND should map to CodeNotFound")
	}
	if errxconnect.CodeFor(errx.New("bug").WithClass(errx.ClassDefect)) != connectrpc.CodeInternal {
		t.Fatal("defect should map to CodeInternal")
	}
	if errxconnect.CodeFor(errx.New("c").WithClass(errx.ClassCancelled)) != connectrpc.CodeCanceled {
		t.Fatal("cancelled should map to CodeCanceled")
	}
}

func TestErrorCarriesIdentityAndRedacts(t *testing.T) {
	e := errx.New("internal detail").
		WithDomain("billing").WithCode("TX_FAIL").
		WithPublic("Please retry").
		With("secret", "leak").WithSafe("order_id", "o-1")

	ce := errxconnect.Error(context.Background(), e)
	if ce.Code() != connectrpc.CodeInvalidArgument {
		t.Fatalf("code = %v", ce.Code())
	}
	if ce.Message() != "Please retry" {
		t.Fatalf("message should be public, got %q", ce.Message())
	}
	var sawInfo bool
	for _, d := range ce.Details() {
		v, err := d.Value()
		if err != nil {
			continue
		}
		if info, ok := v.(*errdetails.ErrorInfo); ok {
			sawInfo = true
			if info.Reason != "TX_FAIL" || info.Domain != "billing" {
				t.Fatalf("identity wrong: %+v", info)
			}
			if _, leaked := info.Metadata["secret"]; leaked {
				t.Fatal("unsafe field leaked into ErrorInfo")
			}
			if info.Metadata["order_id"] != "o-1" {
				t.Fatalf("safe field missing: %+v", info.Metadata)
			}
		}
	}
	if !sawInfo {
		t.Fatal("ErrorInfo detail missing")
	}
}
