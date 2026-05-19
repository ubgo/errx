package connect_test

import (
	"context"
	"testing"

	connectrpc "connectrpc.com/connect"
	"github.com/ubgo/errx"
	errxconnect "github.com/ubgo/errx/contrib/connect"
)

func TestCodeForAllMappingsAndFallbacks(t *testing.T) {
	want := map[string]connectrpc.Code{
		"NOT_FOUND":           connectrpc.CodeNotFound,
		"ALREADY_EXISTS":      connectrpc.CodeAlreadyExists,
		"CONFLICT":            connectrpc.CodeAborted,
		"VALIDATION":          connectrpc.CodeInvalidArgument,
		"INVALID_ARGUMENT":    connectrpc.CodeInvalidArgument,
		"UNAUTHENTICATED":     connectrpc.CodeUnauthenticated,
		"PERMISSION_DENIED":   connectrpc.CodePermissionDenied,
		"FORBIDDEN":           connectrpc.CodePermissionDenied,
		"RESOURCE_EXHAUSTED":  connectrpc.CodeResourceExhausted,
		"FAILED_PRECONDITION": connectrpc.CodeFailedPrecondition,
		"UNAVAILABLE":         connectrpc.CodeUnavailable,
		"UNIMPLEMENTED":       connectrpc.CodeUnimplemented,
		"TIMEOUT":             connectrpc.CodeDeadlineExceeded,
	}
	for code, c := range want {
		if got := errxconnect.CodeFor(errx.New("x").WithCode(code)); got != c {
			t.Fatalf("CodeFor(%s) = %v, want %v", code, got, c)
		}
	}
	if errxconnect.CodeFor(errx.New("x").WithClass(errx.ClassDefect)) != connectrpc.CodeInternal {
		t.Fatal("defect → Internal")
	}
	if errxconnect.CodeFor(errx.New("x").WithClass(errx.ClassCancelled)) != connectrpc.CodeCanceled {
		t.Fatal("cancelled → Canceled")
	}
	if errxconnect.CodeFor(errx.New("x").WithRetryable(0)) != connectrpc.CodeUnavailable {
		t.Fatal("retryable expected → Unavailable")
	}
	if errxconnect.CodeFor(errx.New("x")) != connectrpc.CodeInvalidArgument {
		t.Fatal("plain expected → InvalidArgument")
	}
	if errxconnect.CodeFor(context.Canceled) != connectrpc.CodeUnknown {
		t.Fatal("non-errx → Unknown")
	}
}

func TestErrorVariants(t *testing.T) {
	// Operator-message fallback (no public) + no identity → no detail added.
	ce := errxconnect.Error(context.Background(), errx.New("operator only"))
	if ce.Message() != "operator only" {
		t.Fatalf("message fallback = %q", ce.Message())
	}
	if len(ce.Details()) != 0 {
		t.Fatalf("bare error should have no details: %d", len(ce.Details()))
	}
	// Non-errx → CodeUnknown wrapped.
	ce2 := errxconnect.Error(context.Background(), context.Canceled)
	if ce2.Code() != connectrpc.CodeUnknown {
		t.Fatalf("non-errx code = %v", ce2.Code())
	}
}
