package errx

import "log/slog"

var _ slog.LogValuer = (*Error)(nil)

// LogValue lets the error describe itself to log/slog as a structured group
// — code, domain, class, severity, fingerprint, retryable, and the safe
// fields — instead of being stringified into the message. Unsafe fields are
// redacted, so structured logs are safe by construction.
//
//	slog.Error("request failed", "err", err) // err is *errx.Error
func (e *Error) LogValue() slog.Value {
	if e == nil {
		return slog.StringValue("<nil>")
	}
	attrs := make([]slog.Attr, 0, 8)
	attrs = append(attrs, slog.String("msg", e.Error()))
	if e.domain != "" {
		attrs = append(attrs, slog.String("domain", e.domain))
	}
	if e.code != "" {
		attrs = append(attrs, slog.String("code", e.code))
	}
	attrs = append(attrs,
		slog.String("class", e.class.String()),
		slog.String("severity", e.severity.String()),
		slog.String("fingerprint", e.Fingerprint()),
	)
	if e.retryable {
		attrs = append(attrs, slog.Bool("retryable", true))
		if e.retryAfter > 0 {
			attrs = append(attrs, slog.Duration("retry_after", e.retryAfter))
		}
	}
	if e.traceID != "" {
		attrs = append(attrs, slog.String("trace_id", e.traceID))
	}
	if len(e.fields) > 0 {
		fa := make([]slog.Attr, 0, len(e.fields))
		for _, f := range e.fields {
			if f.Safe {
				fa = append(fa, slog.Any(f.Key, f.Value))
			} else {
				fa = append(fa, slog.String(f.Key, RedactedMarker))
			}
		}
		attrs = append(attrs, slog.Attr{Key: "fields", Value: slog.GroupValue(fa...)})
	}
	return slog.GroupValue(attrs...)
}
