// Package sentry reports errx errors to Sentry. It reuses the core
// deterministic fingerprint so the SAME logical bug groups identically in
// Sentry, Prometheus, logs and on the wire (no server-side guessing). It
// consumes the neutral errx.Report; unsafe fields are already redacted.
package sentry

import (
	"context"

	"github.com/getsentry/sentry-go"
	"github.com/ubgo/errx"
)

// Sink captures errx Reports as Sentry events.
type Sink struct {
	hub *sentry.Hub // optional; falls back to the hub on the context / current hub
}

// New returns a sink that uses the given hub. Pass nil to resolve the hub
// per-call from the context (then sentry.CurrentHub).
func New(hub *sentry.Hub) *Sink { return &Sink{hub: hub} }

// Name identifies the sink.
func (*Sink) Name() string { return "sentry" }

func levelOf(s errx.Severity) sentry.Level {
	switch s {
	case errx.SevDebug:
		return sentry.LevelDebug
	case errx.SevInfo:
		return sentry.LevelInfo
	case errx.SevWarn:
		return sentry.LevelWarning
	case errx.SevFatal:
		return sentry.LevelFatal
	default:
		return sentry.LevelError
	}
}

func (s *Sink) hubFor(ctx context.Context) *sentry.Hub {
	if s.hub != nil {
		return s.hub
	}
	if h := sentry.GetHubFromContext(ctx); h != nil {
		return h
	}
	return sentry.CurrentHub()
}

// Emit sends the report to Sentry: level from severity, code/domain as
// tags, the core fingerprint as the grouping key, safe fields as context,
// mechanism handled=true (it was captured, not a crash), and the errx
// stack as the exception stacktrace.
func (s *Sink) Emit(ctx context.Context, r *errx.Report) error {
	hub := s.hubFor(ctx)
	if hub == nil {
		return nil
	}
	ev := sentry.NewEvent()
	ev.Level = levelOf(r.Severity)
	ev.Message = r.Message
	if r.Fingerprint != "" {
		ev.Fingerprint = []string{r.Fingerprint}
	}
	if ev.Tags == nil {
		ev.Tags = map[string]string{}
	}
	if r.Code != "" {
		ev.Tags["code"] = r.Code
	}
	if r.Domain != "" {
		ev.Tags["domain"] = r.Domain
	}
	ev.Tags["class"] = r.Class.String()
	if r.TraceID != "" {
		ev.Tags["trace_id"] = r.TraceID
	}

	fields := map[string]any{}
	for _, f := range r.Fields {
		fields[f.Key] = f.Value // Snapshot already redacted unsafe values
	}
	if r.Hint != "" {
		fields["hint"] = r.Hint
	}
	if r.Owner != "" {
		fields["owner"] = r.Owner
	}
	if len(fields) > 0 {
		ev.Contexts["errx"] = fields
	}

	ex := sentry.Exception{
		Type:  exceptionType(r),
		Value: r.Message,
		Mechanism: &sentry.Mechanism{
			Type:    "errx",
			Handled: boolPtr(true),
		},
	}
	if len(r.Stack) > 0 {
		st := &sentry.Stacktrace{Frames: make([]sentry.Frame, 0, len(r.Stack))}
		// Sentry expects frames oldest-first; errx stack is innermost-first.
		for i := len(r.Stack) - 1; i >= 0; i-- {
			fr := r.Stack[i]
			st.Frames = append(st.Frames, sentry.Frame{
				Function: fr.Function,
				Filename: fr.File,
				Lineno:   fr.Line,
			})
		}
		ex.Stacktrace = st
	}
	ev.Exception = []sentry.Exception{ex}

	hub.CaptureEvent(ev)
	return nil
}

func exceptionType(r *errx.Report) string {
	switch {
	case r.Domain != "" && r.Code != "":
		return r.Domain + "/" + r.Code
	case r.Code != "":
		return r.Code
	default:
		return "errx.Error"
	}
}

func boolPtr(b bool) *bool { return &b }
