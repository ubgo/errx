package errx

import (
	"context"
	"sync"
)

// ContextExtractor pulls correlation data out of a context.Context. The
// core is stdlib-only and cannot import OpenTelemetry; an otel contrib
// adapter registers an extractor explicitly (no init() magic) so
// Error.Context(ctx) can attach trace/span ids and request-scoped fields.
type ContextExtractor func(ctx context.Context) (traceID, spanID string, fields []Field)

var (
	ctxMu         sync.RWMutex
	ctxExtractors []ContextExtractor
)

// RegisterContextExtractor adds an extractor consulted by Error.Context.
// Call it once at process start (e.g. from an adapter's wiring), never via
// init().
func RegisterContextExtractor(fn ContextExtractor) {
	if fn == nil {
		return
	}
	ctxMu.Lock()
	ctxExtractors = append(ctxExtractors, fn)
	ctxMu.Unlock()
}

// Context enriches the error from ctx via the registered extractors:
// trace/span ids (only if not already set) plus any request-scoped fields.
// No-op when ctx is nil or no extractor is registered.
func (e *Error) Context(ctx context.Context) *Error {
	if e == nil || ctx == nil {
		return e
	}
	ctxMu.RLock()
	exs := ctxExtractors
	ctxMu.RUnlock()
	for _, ex := range exs {
		tid, sid, fs := ex(ctx)
		if tid != "" && e.traceID == "" {
			e.traceID = tid
		}
		if sid != "" && e.spanID == "" {
			e.spanID = sid
		}
		if len(fs) > 0 {
			e.fields = append(e.fields, fs...)
		}
	}
	e.fp = ""
	return e
}
