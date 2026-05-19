package otel_test

import (
	"context"
	"testing"

	"github.com/ubgo/errx"
	otelsink "github.com/ubgo/errx/contrib/otel"
	"go.opentelemetry.io/otel/trace"
)

func TestEmitNoSpanIsNoop(t *testing.T) {
	s := otelsink.NewSink()
	if s.Name() != "otel" {
		t.Fatalf("Name = %q", s.Name())
	}
	rep, _ := errx.Snapshot(context.Background(), errx.New("x").WithCode("C"))
	if err := s.Emit(context.Background(), &rep); err != nil {
		t.Fatalf("Emit with no span should be a no-op, got %v", err)
	}
}

func TestInstallExtractsTraceSpanIDs(t *testing.T) {
	otelsink.Install()

	tid, _ := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	sid, _ := trace.SpanIDFromHex("0123456789abcdef")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: tid,
		SpanID:  sid,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	e := errx.New("op failed").WithCode("X").Context(ctx)
	if e.TraceID() != tid.String() || e.SpanID() != sid.String() {
		t.Fatalf("ids not extracted: trace=%q span=%q", e.TraceID(), e.SpanID())
	}
}
