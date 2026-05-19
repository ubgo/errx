// Package httpx renders an errx error as an RFC 9457 problem+json response.
// It is stdlib-only: it consumes the neutral errx.Report (so unsafe fields
// are already redacted) and never touches *errx.Error internals.
package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/ubgo/errx"
)

// Problem is an RFC 9457 (application/problem+json) document. Extension
// members (code, fingerprint, trace_id, fields) ride alongside the standard
// members; unknown members are ignored by conformant clients.
type Problem struct {
	Type     string `json:"type"`               // URI identifying the problem class
	Title    string `json:"title"`              // stable, human summary
	Status   int    `json:"status"`             // advisory HTTP status
	Detail   string `json:"detail,omitempty"`   // occurrence-specific, user-safe
	Instance string `json:"instance,omitempty"` // URI of this occurrence

	// Extension members.
	Code        string            `json:"code,omitempty"`
	Fingerprint string            `json:"fingerprint,omitempty"`
	TraceID     string            `json:"traceId,omitempty"`
	DocURL      string            `json:"docUrl,omitempty"`
	Localized   map[string]string `json:"localized,omitempty"` // BCP-47 -> message
	Fields      map[string]any    `json:"fields,omitempty"`    // safe fields only
}

// TypeBaseURL is prepended to an error code to form the problem `type` URI
// when no explicit type is registered. "about:blank" is used when there is
// no code (RFC 9457 default).
var TypeBaseURL = "about:blank"

var (
	mu        sync.RWMutex
	statusFor = map[string]int{
		// Conventional code → status. Extend via RegisterStatus.
		"NOT_FOUND":           http.StatusNotFound,
		"ALREADY_EXISTS":      http.StatusConflict,
		"CONFLICT":            http.StatusConflict,
		"VALIDATION":          http.StatusUnprocessableEntity,
		"INVALID_ARGUMENT":    http.StatusBadRequest,
		"UNAUTHENTICATED":     http.StatusUnauthorized,
		"PERMISSION_DENIED":   http.StatusForbidden,
		"FORBIDDEN":           http.StatusForbidden,
		"RESOURCE_EXHAUSTED":  http.StatusTooManyRequests,
		"FAILED_PRECONDITION": http.StatusPreconditionFailed,
		"UNAVAILABLE":         http.StatusServiceUnavailable,
		"UNIMPLEMENTED":       http.StatusNotImplemented,
		"TIMEOUT":             http.StatusGatewayTimeout,
	}
	typeFor = map[string]string{}
)

// RegisterStatus maps an error code to an HTTP status. Explicit wiring, no
// init() magic.
func RegisterStatus(code string, status int) {
	mu.Lock()
	statusFor[code] = status
	mu.Unlock()
}

// RegisterType maps an error code to a problem-type URI.
func RegisterType(code, uri string) {
	mu.Lock()
	typeFor[code] = uri
	mu.Unlock()
}

// StatusFor resolves the HTTP status for err: an explicit code mapping
// first, otherwise derived from the class (defect → 500, cancelled → 499
// client-closed, expected → 400) with a retryable expected error → 503.
func StatusFor(err error) int {
	rep, ok := errx.Snapshot(context.Background(), err)
	if !ok {
		return http.StatusInternalServerError
	}
	mu.RLock()
	s, found := statusFor[rep.Code]
	mu.RUnlock()
	if found {
		return s
	}
	switch rep.Class {
	case errx.ClassDefect:
		return http.StatusInternalServerError
	case errx.ClassCancelled:
		return 499 // client closed request (nginx convention)
	default:
		if rep.Retryable {
			return http.StatusServiceUnavailable
		}
		return http.StatusBadRequest
	}
}

// FromError projects err into a Problem. Returns ok=false if err carries no
// *errx.Error.
func FromError(ctx context.Context, err error) (Problem, bool) {
	rep, ok := errx.Snapshot(ctx, err)
	if !ok {
		return Problem{}, false
	}
	status := StatusFor(err)
	title := rep.Code
	if title == "" {
		title = http.StatusText(status)
	}
	detail := rep.Public
	if detail == "" {
		detail = http.StatusText(status)
	}
	p := Problem{
		Type:        problemType(rep.Code),
		Title:       title,
		Status:      status,
		Detail:      detail,
		Code:        rep.Code,
		Fingerprint: rep.Fingerprint,
		TraceID:     rep.TraceID,
		DocURL:      rep.DocURL,
		Localized:   rep.Localized,
	}
	for _, f := range rep.Fields {
		if f.Safe { // Snapshot already redacted unsafe values
			if p.Fields == nil {
				p.Fields = map[string]any{}
			}
			p.Fields[f.Key] = f.Value
		}
	}
	return p, true
}

func problemType(code string) string {
	if code == "" {
		return "about:blank"
	}
	mu.RLock()
	u, ok := typeFor[code]
	mu.RUnlock()
	if ok {
		return u
	}
	if TypeBaseURL == "about:blank" {
		return "about:blank"
	}
	return TypeBaseURL + code
}

// Write renders err as an application/problem+json response. If err is not
// an errx error it falls back to a 500 problem. Returns the status written.
func Write(w http.ResponseWriter, r *http.Request, err error) int {
	ctx := context.Background()
	if r != nil {
		ctx = r.Context()
	}
	p, ok := FromError(ctx, err)
	if !ok {
		p = Problem{Type: "about:blank", Title: "Internal Server Error", Status: http.StatusInternalServerError, Detail: "Internal Server Error"}
	}
	if r != nil && r.URL != nil {
		p.Instance = r.URL.Path
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
	return p.Status
}
