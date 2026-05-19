// Package errx is a structured error type for Go: one coherent value carrying
// a stable machine identity (domain + code), an expected/defect/cancelled
// class, a separate operator and end-user message, typed structured fields
// with per-field redaction, retry and severity metadata, a lazily-captured
// stack, suppressed (secondary) errors, and a deterministic fingerprint —
// while staying a drop-in over the standard library's errors.Is/As/Join.
//
// The core (this package) imports nothing beyond the standard library.
// Observability and transport integrations live under contrib/* as separate
// modules that consume the neutral Report DTO via Snapshot; the core never
// imports them and there is no init() magic — registration is explicit.
package errx

import (
	"time"
)

// Class is the coarse handling category of an error. Middleware and
// observability layers branch on it: return 4xx + maybe retry for Expected,
// alert + 5xx for Defect, stay quiet for Cancelled.
type Class uint8

const (
	// ClassExpected is a normal, anticipated failure the caller can handle
	// (validation failed, not found, conflict). The default.
	ClassExpected Class = iota
	// ClassDefect is a bug or invariant violation — it should never happen
	// and warrants an alert, not a user-facing 4xx.
	ClassDefect
	// ClassCancelled is work abandoned because the caller went away
	// (context cancelled / deadline exceeded). Usually not worth logging loudly.
	ClassCancelled
)

func (c Class) String() string {
	switch c {
	case ClassExpected:
		return "expected"
	case ClassDefect:
		return "defect"
	case ClassCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// Severity is the log/alert level carried by an error so sinks route it
// without re-deriving it from the message.
type Severity uint8

const (
	SevDebug Severity = iota
	SevInfo
	SevWarn
	SevError // default
	SevFatal
)

func (s Severity) String() string {
	switch s {
	case SevDebug:
		return "debug"
	case SevInfo:
		return "info"
	case SevWarn:
		return "warn"
	case SevError:
		return "error"
	case SevFatal:
		return "fatal"
	default:
		return "error"
	}
}

// Field is one structured key/value attached to an error. Safe marks the
// value as non-sensitive; everything is treated as unsafe (redacted at a
// trust boundary) unless explicitly marked Safe — safe-by-default redaction
// without the redactable-string ergonomic tax.
type Field struct {
	Key   string
	Value any
	Safe  bool
}

// Error is the core structured error value. It is an ordinary Go error;
// generics are never required to use it.
type Error struct {
	domain    string
	code      string
	class     Class
	severity  Severity
	msg       string // operator/internal message
	publicMsg string // end-user-safe message
	hint      string // remediation guidance
	owner     string // responsible team, for routing/alerting

	fields     []Field
	cause      error
	suppressed []error // secondary errors (e.g. a failed Close during unwind)

	retryable  bool
	retryAfter time.Duration

	traceID string
	spanID  string

	// Optional diagnostic layer (miette-style). Rendered by the diag
	// subpackage; absent on most errors and zero-cost when unused.
	docURL    string
	srcName   string
	src       string
	labels    []Label
	localized map[string]string // BCP-47 locale -> end-user message

	opaque    bool      // when true, Unwrap returns nil (a barrier)
	fromPanic bool      // captured from a panic via Recover
	wrapPC    uintptr   // caller site where this layer wrapped (return-trace)
	pcs       []uintptr // lazily-symbolized origin stack
	decoded   []Frame   // pre-symbolized stack from the wire (Decode); used when pcs is empty
	fp        string    // cached fingerprint
}

// Error implements the error interface, returning the operator message.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.cause != nil && !e.opaque {
		if e.msg == "" {
			return e.cause.Error()
		}
		return e.msg + ": " + e.cause.Error()
	}
	return e.msg
}

// Unwrap exposes the wrapped cause plus any suppressed errors so
// errors.Is/As traverse the whole chain (Go 1.20 multi-unwrap form).
// An Opaque() error returns nil here — a deliberate barrier that hides a
// dependency's sentinels from callers. Note: like errors.Join, the single
// errors.Unwrap returns nil for this type; use Cause() for the primary cause.
func (e *Error) Unwrap() []error {
	if e == nil || e.opaque {
		return nil
	}
	if e.cause == nil && len(e.suppressed) == 0 {
		return nil
	}
	out := make([]error, 0, 1+len(e.suppressed))
	if e.cause != nil {
		out = append(out, e.cause)
	}
	out = append(out, e.suppressed...)
	return out
}

// ---- accessors -------------------------------------------------------------

func (e *Error) Domain() string            { return e.domain }
func (e *Error) Code() string              { return e.code }
func (e *Error) ClassOf() Class            { return e.class }
func (e *Error) SeverityOf() Severity      { return e.severity }
func (e *Error) Hint() string              { return e.hint }
func (e *Error) Owner() string             { return e.owner }
func (e *Error) Cause() error              { return e.cause }
func (e *Error) Suppressed() []error       { return e.suppressed }
func (e *Error) Retryable() bool           { return e.retryable }
func (e *Error) RetryAfter() time.Duration { return e.retryAfter }
func (e *Error) TraceID() string           { return e.traceID }
func (e *Error) SpanID() string            { return e.spanID }

// Public returns the end-user-safe message, falling back to fallback when
// none was set so callers never accidentally surface the operator message.
func (e *Error) Public(fallback string) string {
	if e == nil {
		return fallback
	}
	if e.publicMsg != "" {
		return e.publicMsg
	}
	return fallback
}

// Fields returns a copy of the structured fields (defensive).
func (e *Error) Fields() []Field {
	if len(e.fields) == 0 {
		return nil
	}
	out := make([]Field, len(e.fields))
	copy(out, e.fields)
	return out
}

// ---- fluent builders (mutate-and-return; build then return) ----------------

func (e *Error) WithDomain(d string) *Error     { e.domain = d; e.fp = ""; return e }
func (e *Error) WithCode(c string) *Error       { e.code = c; e.fp = ""; return e }
func (e *Error) WithClass(c Class) *Error       { e.class = c; return e }
func (e *Error) WithSeverity(s Severity) *Error { e.severity = s; return e }
func (e *Error) WithPublic(m string) *Error     { e.publicMsg = m; return e }
func (e *Error) WithHint(h string) *Error       { e.hint = h; return e }
func (e *Error) WithOwner(o string) *Error      { e.owner = o; return e }
func (e *Error) WithTrace(traceID, spanID string) *Error {
	e.traceID, e.spanID = traceID, spanID
	return e
}

// WithRetryable marks the error retryable and, optionally, after how long.
func (e *Error) WithRetryable(after time.Duration) *Error {
	e.retryable = true
	e.retryAfter = after
	return e
}

// With attaches an unsafe (redacted at a trust boundary) structured field.
func (e *Error) With(key string, value any) *Error {
	e.fields = append(e.fields, Field{Key: key, Value: value})
	return e
}

// WithSafe attaches a field explicitly marked non-sensitive (never redacted).
func (e *Error) WithSafe(key string, value any) *Error {
	e.fields = append(e.fields, Field{Key: key, Value: value, Safe: true})
	return e
}

// Suppress records a secondary error (a cleanup/Close failure during error
// unwind) alongside the primary, instead of one masking the other. The
// suppressed errors remain matchable via errors.Is/As.
func (e *Error) Suppress(errs ...error) *Error {
	for _, s := range errs {
		if s != nil {
			e.suppressed = append(e.suppressed, s)
		}
	}
	return e
}

// Opaque turns this error into a barrier: errors.Is/As will not see the
// wrapped cause. Use deliberately to stop a dependency's sentinels leaking
// into your package's API surface.
func (e *Error) Opaque() *Error { e.opaque = true; return e }

// Label is a source span annotation: [Start, Start+Len) into the source
// set by WithSource, with a short message rendered under a caret underline.
type Label struct {
	Start int
	Len   int
	Msg   string
}

// WithURL attaches a documentation URL for this error's code, rendered as
// a clickable reference by the diag package (miette's url(docsrs) idea).
func (e *Error) WithURL(url string) *Error { e.docURL = url; return e }

// URL returns the documentation URL: the one set via WithURL, else the
// one registered for this error's code (see RegisterDoc), else "".
func (e *Error) URL() string {
	if e == nil {
		return ""
	}
	if e.docURL != "" {
		return e.docURL
	}
	if d, ok := e.resolveDoc(); ok {
		return d.URL
	}
	return ""
}

// Remediation returns operator guidance: the Hint set on the error, else
// the remediation registered for its code.
func (e *Error) Remediation() string {
	if e == nil {
		return ""
	}
	if e.hint != "" {
		return e.hint
	}
	if d, ok := e.resolveDoc(); ok {
		return d.Remediation
	}
	return ""
}

// WithSource attaches the source text that diagnostic labels index into
// (a query, a config file, a DSL snippet — whatever produced the error).
func (e *Error) WithSource(name, content string) *Error {
	e.srcName, e.src = name, content
	return e
}

// WithLabel adds a labeled span over the source set by WithSource.
func (e *Error) WithLabel(start, length int, msg string) *Error {
	e.labels = append(e.labels, Label{Start: start, Len: length, Msg: msg})
	return e
}

// WithLocalized attaches an end-user message for a BCP-47 locale
// (e.g. "fr-FR"). The locale-free Public message remains the default.
func (e *Error) WithLocalized(locale, msg string) *Error {
	if e.localized == nil {
		e.localized = map[string]string{}
	}
	e.localized[locale] = msg
	return e
}

// Localized returns the message for locale, falling back to a same-language
// match (e.g. "fr" for "fr-CA"), then the Public message, then fallback.
func (e *Error) Localized(locale, fallback string) string {
	if e == nil {
		return fallback
	}
	if m, ok := e.localized[locale]; ok {
		return m
	}
	if i := indexByte(locale, '-'); i > 0 {
		lang := locale[:i]
		for k, v := range e.localized {
			if k == lang || (len(k) > len(lang) && k[:len(lang)] == lang && k[len(lang)] == '-') {
				return v
			}
		}
	}
	return e.Public(fallback)
}

// LocaleMessages returns a copy of all locale-specific messages.
func (e *Error) LocaleMessages() map[string]string {
	if len(e.localized) == 0 {
		return nil
	}
	out := make(map[string]string, len(e.localized))
	for k, v := range e.localized {
		out[k] = v
	}
	return out
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// Source returns the diagnostic source name and content.
func (e *Error) Source() (name, content string) { return e.srcName, e.src }

// Labels returns a copy of the diagnostic labels.
func (e *Error) Labels() []Label {
	if len(e.labels) == 0 {
		return nil
	}
	out := make([]Label, len(e.labels))
	copy(out, e.labels)
	return out
}
