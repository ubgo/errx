package errx

import (
	"context"
	"sync"
)

// Report is the neutral, serializable projection of an *Error. Every
// contrib adapter (sentry, otel, http, grpc, prometheus, ...) consumes a
// Report and nothing else — adapters never touch *Error internals and the
// core never imports an adapter. This is the single seam (ERR-ADR-001).
//
// Unsafe fields are redacted here, so any sink that crosses a trust
// boundary is safe by construction.
type Report struct {
	Domain      string
	Code        string
	Class       Class
	Severity    Severity
	Message     string // operator message
	Public      string // end-user-safe message
	Hint        string
	Owner       string
	DocURL      string
	Localized   map[string]string // BCP-47 locale -> message
	Fingerprint string
	Retryable   bool
	RetryAfter  string // human duration, "" if none
	TraceID     string
	SpanID      string
	Fields      []Field // unsafe values already replaced with redacted marker
	Stack       []Frame // origin stack
	Trace       []Frame // return-trace: where the error traveled (per-wrap)
	Suppressed  []string
	Cause       string
}

// RedactedMarker replaces the value of any field not marked Safe when the
// error is projected to a Report.
const RedactedMarker = "‹redacted›"

// Snapshot projects the nearest *Error in err into a Report, resolving the
// lazy stack and redacting unsafe fields. ctx is accepted for future
// trace-extraction hooks; nil is fine.
func Snapshot(ctx context.Context, err error) (Report, bool) {
	_ = ctx
	e := Get(err)
	if e == nil {
		return Report{}, false
	}
	r := Report{
		Domain:      e.domain,
		Code:        e.code,
		Class:       e.class,
		Severity:    e.severity,
		Message:     e.Error(),
		Public:      e.publicMsg,
		Hint:        e.Remediation(),
		Owner:       e.owner,
		DocURL:      e.URL(),
		Localized:   e.LocaleMessages(),
		Fingerprint: e.Fingerprint(),
		Retryable:   e.retryable,
		TraceID:     e.traceID,
		SpanID:      e.spanID,
		Stack:       e.Frames(),
		Trace:       e.Trace(),
	}
	if e.retryAfter > 0 {
		r.RetryAfter = e.retryAfter.String()
	}
	if e.cause != nil {
		r.Cause = e.cause.Error()
	}
	for _, f := range e.fields {
		if f.Safe {
			r.Fields = append(r.Fields, f)
		} else {
			r.Fields = append(r.Fields, Field{Key: f.Key, Value: RedactedMarker, Safe: false})
		}
	}
	for _, s := range e.suppressed {
		r.Suppressed = append(r.Suppressed, s.Error())
	}
	return r, true
}

// Sink receives projected error Reports. Implementations live in contrib/*.
type Sink interface {
	Name() string
	Emit(ctx context.Context, r *Report) error
}

// Registry fans a Report out to registered sinks. Registration is explicit
// (caller wires it) — there is no init() magic. Per-sink Emit errors are
// swallowed so one bad sink cannot break reporting; SinkError records them
// for observability of the observability.
type Registry struct {
	mu          sync.RWMutex
	sinks       []Sink
	OnSinkError func(name string, err error)
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry { return &Registry{} }

// Add registers a sink. Returns the registry for chaining.
func (r *Registry) Add(s Sink) *Registry {
	r.mu.Lock()
	r.sinks = append(r.sinks, s)
	r.mu.Unlock()
	return r
}

// Report projects err and emits it to every sink. No-op if err carries no
// *Error. Safe for concurrent use.
func (r *Registry) Report(ctx context.Context, err error) {
	rep, ok := Snapshot(ctx, err)
	if !ok {
		return
	}
	r.mu.RLock()
	sinks := r.sinks
	r.mu.RUnlock()
	for _, s := range sinks {
		if e := s.Emit(ctx, &rep); e != nil && r.OnSinkError != nil {
			r.OnSinkError(s.Name(), e)
		}
	}
}
