// Package goerr is a migration shim between the shipped github.com/ubgo/goerr
// and github.com/ubgo/errx. It lets a codebase adopt errx incrementally:
// convert inbound goerr values to *errx.Error (From), or hand an errx error
// back to code still expecting goerr (To).
package goerr

import (
	"errors"

	"github.com/ubgo/errx"
	gerr "github.com/ubgo/goerr"
)

func severityFromLevel(level string) errx.Severity {
	switch level {
	case "debug":
		return errx.SevDebug
	case "info":
		return errx.SevInfo
	case "warn", "warning":
		return errx.SevWarn
	case "fatal", "critical":
		return errx.SevFatal
	default:
		return errx.SevError
	}
}

func levelFromSeverity(s errx.Severity) string {
	switch s {
	case errx.SevDebug:
		return "debug"
	case errx.SevInfo:
		return "info"
	case errx.SevWarn:
		return "warn"
	case errx.SevFatal:
		return "fatal"
	default:
		return "error"
	}
}

// From converts a *goerr.Error (found anywhere in err's chain) into an
// *errx.Error, preserving the operator/user messages, code, trace id,
// severity and key/values. KV and Data become unsafe fields (redacted at a
// trust boundary by default — opt back in with errx WithSafe if needed).
// Returns nil if err is nil; if no goerr is present, err is wrapped as-is.
func From(err error) *errx.Error {
	if err == nil {
		return nil
	}
	var g *gerr.Error
	if !errors.As(err, &g) {
		return errx.Wrap(err, "")
	}
	r := g.Response()
	e := errx.New(r.Message).
		WithCode(r.Code).
		WithPublic(r.MessageUser).
		WithSeverity(severityFromLevel(r.Level))
	if r.TraceID != "" {
		e.WithTrace(r.TraceID, "")
	}
	for k, v := range r.KV {
		e.With(k, v)
	}
	if r.Data != nil {
		e.With("data", r.Data)
	}
	return e
}

// To converts an errx error into a *goerr.Error so code still written
// against goerr keeps working during migration. Safe fields become KV;
// unsafe fields are dropped (never leak through the bridge).
func To(err error) *gerr.Error {
	if err == nil {
		return nil
	}
	e := errx.Get(err)
	if e == nil {
		return gerr.New(err.Error())
	}
	opts := []gerr.ErrorOption{
		gerr.WithCode(e.Code()),
		gerr.WithMessageUser(e.Public(e.Error())),
		gerr.WithLevel(levelFromSeverity(e.SeverityOf())),
	}
	if e.TraceID() != "" {
		opts = append(opts, gerr.WithTraceID(e.TraceID()))
	}
	kv := map[string]any{}
	for _, f := range e.Fields() {
		if f.Safe {
			kv[f.Key] = f.Value
		}
	}
	if len(kv) > 0 {
		opts = append(opts, gerr.WithKVMap(kv))
	}
	return gerr.New(e.Error(), opts...)
}
