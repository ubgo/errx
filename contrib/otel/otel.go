// Package otel bridges errx errors to OpenTelemetry: it records the error
// as a span event using the exception.* semantic conventions, sets the span
// status to Error, and (via Install) lets errx auto-attach trace/span ids
// from a context. It consumes the neutral errx.Report; it never touches
// *errx.Error internals and there is no init() — wiring is explicit.
package otel

import (
	"context"

	"github.com/ubgo/errx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Sink records errx Reports onto the active span. Register it with an
// errx.Registry:
//
//	reg := errx.NewRegistry().Add(otel.NewSink())
//	reg.Report(ctx, err)
type Sink struct{}

// NewSink returns an OpenTelemetry sink.
func NewSink() *Sink { return &Sink{} }

// Name identifies the sink.
func (*Sink) Name() string { return "otel" }

// Emit records the report on the span in ctx (if any is recording),
// following the exception.* semconv, and sets the span status to Error for
// non-cancelled classes. No-op when no span is recording.
func (*Sink) Emit(ctx context.Context, r *errx.Report) error {
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.IsRecording() {
		return nil
	}
	attrs := []attribute.KeyValue{
		attribute.String("exception.type", exceptionType(r)),
		attribute.String("exception.message", r.Message),
	}
	if len(r.Stack) > 0 {
		attrs = append(attrs, attribute.String("exception.stacktrace", stackString(r.Stack)))
	}
	if r.Code != "" {
		attrs = append(attrs, attribute.String("error.code", r.Code))
	}
	if r.Domain != "" {
		attrs = append(attrs, attribute.String("error.domain", r.Domain))
	}
	if r.Fingerprint != "" {
		attrs = append(attrs, attribute.String("error.fingerprint", r.Fingerprint))
	}
	span.AddEvent("exception", trace.WithAttributes(attrs...))

	// RecordError records the event but does NOT set status — set it too,
	// except for caller-cancelled work which is not a server fault.
	if r.Class != errx.ClassCancelled {
		span.SetStatus(codes.Error, r.Message)
	}
	return nil
}

func exceptionType(r *errx.Report) string {
	if r.Code != "" {
		if r.Domain != "" {
			return r.Domain + "/" + r.Code
		}
		return r.Code
	}
	return "errx.Error"
}

func stackString(frames []errx.Frame) string {
	b := make([]byte, 0, 64*len(frames))
	for _, f := range frames {
		b = append(b, f.Function...)
		b = append(b, '\n', '\t')
		b = append(b, f.File...)
		b = append(b, ':')
		b = appendInt(b, f.Line)
		b = append(b, '\n')
	}
	return string(b)
}

func appendInt(b []byte, n int) []byte {
	if n == 0 {
		return append(b, '0')
	}
	var tmp [20]byte
	i := len(tmp)
	for n > 0 {
		i--
		tmp[i] = byte('0' + n%10)
		n /= 10
	}
	return append(b, tmp[i:]...)
}

// Install registers an errx context extractor so errx.Error.Context(ctx)
// picks up the active OpenTelemetry trace and span ids. Call once at
// process start.
func Install() {
	errx.RegisterContextExtractor(func(ctx context.Context) (string, string, []errx.Field) {
		sc := trace.SpanContextFromContext(ctx)
		if !sc.IsValid() {
			return "", "", nil
		}
		return sc.TraceID().String(), sc.SpanID().String(), nil
	})
}
