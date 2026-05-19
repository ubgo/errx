package otel_test

import (
	"context"
	"testing"

	"github.com/ubgo/errx"
	otelsink "github.com/ubgo/errx/contrib/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func newRecorder() (*tracetest.SpanRecorder, *sdktrace.TracerProvider) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	return sr, tp
}

func TestEmitRecordsExceptionAndSetsStatus(t *testing.T) {
	sr, tp := newRecorder()
	tr := tp.Tracer("test")
	ctx, span := tr.Start(context.Background(), "op")

	e := errx.New("pq: deadlock on user_orders").
		WithDomain("billing").WithCode("TX_FAIL")
	rep, _ := errx.Snapshot(ctx, e) // captures a real origin stack with frames
	if err := otelsink.NewSink().Emit(ctx, &rep); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	span.End()

	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("want 1 span, got %d", len(spans))
	}
	s := spans[0]
	if s.Status().Code != codes.Error {
		t.Fatalf("span status = %v, want Error", s.Status().Code)
	}
	var ev *string
	for _, e := range s.Events() {
		if e.Name == "exception" {
			name := e.Name
			ev = &name
			var sawType, sawMsg, sawStack, sawCode bool
			for _, a := range e.Attributes {
				switch string(a.Key) {
				case "exception.type":
					sawType = a.Value.AsString() == "billing/TX_FAIL"
				case "exception.message":
					sawMsg = a.Value.AsString() != ""
				case "exception.stacktrace":
					sawStack = a.Value.AsString() != "" // exercises stackString/appendInt
				case "error.code":
					sawCode = a.Value.AsString() == "TX_FAIL"
				}
			}
			if !sawType || !sawMsg || !sawStack || !sawCode {
				t.Fatalf("exception attrs incomplete: type=%v msg=%v stack=%v code=%v",
					sawType, sawMsg, sawStack, sawCode)
			}
		}
	}
	if ev == nil {
		t.Fatal("no exception event recorded")
	}
}

func TestEmitCancelledDoesNotSetErrorStatus(t *testing.T) {
	sr, tp := newRecorder()
	ctx, span := tp.Tracer("t").Start(context.Background(), "op")

	e := errx.New("client went away").WithClass(errx.ClassCancelled).WithCode("CANCELLED")
	rep, _ := errx.Snapshot(ctx, e)
	_ = otelsink.NewSink().Emit(ctx, &rep)
	span.End()

	s := sr.Ended()[0]
	if s.Status().Code == codes.Error {
		t.Fatal("cancelled-class error must NOT set span status to Error")
	}
	// The exception event is still recorded.
	found := false
	for _, ev := range s.Events() {
		if ev.Name == "exception" {
			found = true
		}
	}
	if !found {
		t.Fatal("exception event should still be recorded for cancelled")
	}
}
