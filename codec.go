package errx

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
)

var (
	migMu   sync.RWMutex
	codeMig = map[string]string{}
)

// RegisterCodeMigration maps a retired error code to its replacement.
// Decode rewrites old to new, so a newer service still recognises errors
// minted by an older one (and vice versa once both register the pair) —
// the cross-version-skew tolerance cockroachdb/errors gets from
// RegisterTypeMigration, here for the JSON wire form. Explicit, no init().
func RegisterCodeMigration(old, new string) {
	migMu.Lock()
	codeMig[old] = new
	migMu.Unlock()
}

func migrateCode(code string) string {
	if code == "" {
		return code
	}
	migMu.RLock()
	defer migMu.RUnlock()
	// Follow a short chain (A->B->C) with a cycle guard.
	for i := 0; i < 8; i++ {
		n, ok := codeMig[code]
		if !ok || n == code {
			break
		}
		code = n
	}
	return code
}

// wire is the JSON projection used to ship an *Error across a service
// boundary and reconstruct a typed *Error on the far side. It is lossy by
// design: the cause and suppressed errors travel as messages (the remote
// concrete types are not importable here), but identity — domain, code,
// class, severity, retry metadata, fingerprint — survives, so errx.Code /
// HasCode / IsRetryable / ClassOf keep working after the hop. Only Safe
// fields cross; unsafe values are dropped, not redacted-in-place, so
// nothing sensitive ever leaves the process.
type wire struct {
	Domain     string   `json:"domain,omitempty"`
	Code       string   `json:"code,omitempty"`
	Class      Class    `json:"class"`
	Severity   Severity `json:"severity"`
	Msg        string   `json:"msg,omitempty"`
	Public     string   `json:"public,omitempty"`
	Hint       string   `json:"hint,omitempty"`
	Owner      string   `json:"owner,omitempty"`
	Retryable  bool     `json:"retryable,omitempty"`
	RetryAfter string   `json:"retryAfter,omitempty"`
	TraceID    string   `json:"traceId,omitempty"`
	SpanID     string   `json:"spanId,omitempty"`
	Fields     []Field  `json:"fields,omitempty"` // safe fields only
	Stack      []Frame  `json:"stack,omitempty"`
	Suppressed []string `json:"suppressed,omitempty"`
	Cause      string   `json:"cause,omitempty"`
	Fp         string   `json:"fingerprint,omitempty"`
}

// MarshalJSON implements json.Marshaler so json.Marshal(err) yields the
// wire form. Unsafe fields are omitted entirely (never serialized).
func (e *Error) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	w := wire{
		Domain:    e.domain,
		Code:      e.code,
		Class:     e.class,
		Severity:  e.severity,
		Msg:       e.msg,
		Public:    e.publicMsg,
		Hint:      e.hint,
		Owner:     e.owner,
		Retryable: e.retryable,
		TraceID:   e.traceID,
		SpanID:    e.spanID,
		Stack:     e.Frames(),
		Fp:        e.Fingerprint(),
	}
	if e.retryAfter > 0 {
		w.RetryAfter = e.retryAfter.String()
	}
	for _, f := range e.fields {
		if f.Safe {
			w.Fields = append(w.Fields, f)
		}
	}
	if e.cause != nil {
		w.Cause = e.cause.Error()
	}
	for _, s := range e.suppressed {
		w.Suppressed = append(w.Suppressed, s.Error())
	}
	return json.Marshal(w)
}

// Encode serializes err to the JSON wire form. err should contain an
// *Error; otherwise a minimal envelope with just the message is produced.
func Encode(err error) ([]byte, error) {
	if e := Get(err); e != nil {
		return json.Marshal(e)
	}
	if err == nil {
		return []byte("null"), nil
	}
	return json.Marshal(wire{Msg: err.Error(), Class: ClassExpected, Severity: SevError})
}

// Decode reconstructs a typed *Error from the wire form. The cause and any
// suppressed errors come back as opaque errors.New values carrying their
// original messages (remote concrete types are not importable), but the
// reconstructed error keeps its identity so errx.Code / HasCode /
// IsRetryable / ClassOf behave the same across the boundary.
func Decode(data []byte) (*Error, error) {
	var w wire
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}
	e := &Error{
		domain:    w.Domain,
		code:      migrateCode(w.Code),
		class:     w.Class,
		severity:  w.Severity,
		msg:       w.Msg,
		publicMsg: w.Public,
		hint:      w.Hint,
		owner:     w.Owner,
		retryable: w.Retryable,
		traceID:   w.TraceID,
		spanID:    w.SpanID,
		fields:    w.Fields,
		decoded:   w.Stack,
		fp:        w.Fp,
	}
	if w.RetryAfter != "" {
		if d, err := time.ParseDuration(w.RetryAfter); err == nil {
			e.retryAfter = d
		}
	}
	if w.Cause != "" {
		e.cause = errors.New(w.Cause)
	}
	for _, s := range w.Suppressed {
		e.suppressed = append(e.suppressed, errors.New(s))
	}
	return e, nil
}
