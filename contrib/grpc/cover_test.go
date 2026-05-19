package grpc_test

import (
	"context"
	"testing"

	"github.com/ubgo/errx"
	errxgrpc "github.com/ubgo/errx/contrib/grpc"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
)

func TestCodeForAllMappingsAndFallbacks(t *testing.T) {
	want := map[string]codes.Code{
		"NOT_FOUND":           codes.NotFound,
		"ALREADY_EXISTS":      codes.AlreadyExists,
		"CONFLICT":            codes.Aborted,
		"VALIDATION":          codes.InvalidArgument,
		"INVALID_ARGUMENT":    codes.InvalidArgument,
		"UNAUTHENTICATED":     codes.Unauthenticated,
		"PERMISSION_DENIED":   codes.PermissionDenied,
		"FORBIDDEN":           codes.PermissionDenied,
		"RESOURCE_EXHAUSTED":  codes.ResourceExhausted,
		"FAILED_PRECONDITION": codes.FailedPrecondition,
		"UNAVAILABLE":         codes.Unavailable,
		"UNIMPLEMENTED":       codes.Unimplemented,
		"TIMEOUT":             codes.DeadlineExceeded,
	}
	for code, c := range want {
		if got := errxgrpc.CodeFor(errx.New("x").WithCode(code)); got != c {
			t.Fatalf("CodeFor(%s) = %v, want %v", code, got, c)
		}
	}
	// Class fallbacks (no code mapping).
	if errxgrpc.CodeFor(errx.New("x").WithClass(errx.ClassDefect)) != codes.Internal {
		t.Fatal("defect → Internal")
	}
	if errxgrpc.CodeFor(errx.New("x").WithClass(errx.ClassCancelled)) != codes.Canceled {
		t.Fatal("cancelled → Canceled")
	}
	if errxgrpc.CodeFor(errx.New("x").WithRetryable(0)) != codes.Unavailable {
		t.Fatal("retryable expected → Unavailable")
	}
	if errxgrpc.CodeFor(errx.New("x")) != codes.InvalidArgument {
		t.Fatal("plain expected → InvalidArgument")
	}
	if errxgrpc.CodeFor(nil) != codes.Unknown {
		t.Fatal("nil → Unknown")
	}
}

func TestStatusMinimalAndOperatorMessageFallback(t *testing.T) {
	// No public message → status message falls back to operator message;
	// no hint/url/retry/localized → only ErrorInfo detail.
	st := errxgrpc.Status(context.Background(),
		errx.New("operator only").WithDomain("d").WithCode("C"))
	if st.Message() != "operator only" {
		t.Fatalf("message fallback = %q", st.Message())
	}
	var sawInfo, sawOther bool
	for _, d := range st.Details() {
		switch d.(type) {
		case *errdetails.ErrorInfo:
			sawInfo = true
		default:
			sawOther = true
		}
	}
	if !sawInfo || sawOther {
		t.Fatalf("expected only ErrorInfo: info=%v other=%v", sawInfo, sawOther)
	}

	// No identity at all → no details, message is operator message.
	st2 := errxgrpc.Status(context.Background(), errx.New("bare"))
	if len(st2.Details()) != 0 {
		t.Fatalf("bare error should have no details: %d", len(st2.Details()))
	}

	// Non-errx → Unknown.
	if errxgrpc.Status(context.Background(), context.Canceled).Code() == codes.OK {
		t.Fatal("non-errx should not be OK")
	}
}
