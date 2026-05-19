package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/ubgo/errx"
	errxgrpc "github.com/ubgo/errx/contrib/grpc"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCodeMapping(t *testing.T) {
	cases := map[string]codes.Code{
		"NOT_FOUND":         codes.NotFound,
		"VALIDATION":        codes.InvalidArgument,
		"PERMISSION_DENIED": codes.PermissionDenied,
	}
	for code, want := range cases {
		if got := errxgrpc.CodeFor(errx.New("x").WithCode(code)); got != want {
			t.Fatalf("CodeFor(%s) = %v, want %v", code, got, want)
		}
	}
	if errxgrpc.CodeFor(errx.New("bug").WithClass(errx.ClassDefect)) != codes.Internal {
		t.Fatal("defect should map to Internal")
	}
}

func TestStatusCarriesIdentityAndRedacts(t *testing.T) {
	e := errx.New("internal pq deadlock").
		WithDomain("billing").WithCode("TX_FAIL").
		WithPublic("Please retry").
		WithHint("use backoff").
		WithRetryable(2*time.Second).
		With("secret", "leak").WithSafe("order_id", "o-9")

	st := errxgrpc.Status(context.Background(), e)
	// Retryable expected error with no explicit code mapping → Unavailable.
	if st.Code() != codes.Unavailable {
		t.Fatalf("code = %v, want Unavailable", st.Code())
	}
	if st.Message() != "Please retry" {
		t.Fatalf("message should be the public message, got %q", st.Message())
	}

	var sawErrInfo, sawRetry, sawLocalized bool
	for _, d := range st.Details() {
		switch v := d.(type) {
		case *errdetails.ErrorInfo:
			sawErrInfo = true
			if v.Reason != "TX_FAIL" || v.Domain != "billing" {
				t.Fatalf("ErrorInfo identity wrong: %+v", v)
			}
			if _, leaked := v.Metadata["secret"]; leaked {
				t.Fatal("unsafe field leaked into ErrorInfo metadata")
			}
			if v.Metadata["order_id"] != "o-9" {
				t.Fatalf("safe field missing: %+v", v.Metadata)
			}
		case *errdetails.RetryInfo:
			sawRetry = true
			if v.RetryDelay.AsDuration() != 2*time.Second {
				t.Fatalf("retry delay = %v", v.RetryDelay.AsDuration())
			}
		case *errdetails.LocalizedMessage:
			sawLocalized = true
		}
	}
	if !sawErrInfo || !sawRetry || !sawLocalized {
		t.Fatalf("details missing: errinfo=%v retry=%v localized=%v", sawErrInfo, sawRetry, sawLocalized)
	}
}

func TestErrRoundTripsThroughStatus(t *testing.T) {
	e := errx.New("x").WithCode("NOT_FOUND").WithPublic("missing")
	err := errxgrpc.Err(context.Background(), e)
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Fatalf("round trip failed: ok=%v code=%v", ok, st.Code())
	}
	if errxgrpc.Err(context.Background(), nil) != nil {
		t.Fatal("Err(nil) must be nil")
	}
}
