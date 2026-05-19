package errx

import "sync"

// DocEntry is the documentation for an error code: a stable doc URL, a
// one-line summary, and operator remediation guidance. Registering codes
// centrally means an error only needs to carry its code — the human-facing
// docs/URL/remediation are resolved from the registry (miette's url() idea
// + a code catalog), and stay consistent everywhere the code appears.
type DocEntry struct {
	URL         string
	Summary     string
	Remediation string
}

var (
	docMu  sync.RWMutex
	docReg = map[string]DocEntry{}
)

// RegisterDoc records documentation for an error code. Explicit
// registration (call once at startup); no init() magic.
func RegisterDoc(code string, d DocEntry) {
	docMu.Lock()
	docReg[code] = d
	docMu.Unlock()
}

// DocFor returns the registered documentation for a code.
func DocFor(code string) (DocEntry, bool) {
	docMu.RLock()
	d, ok := docReg[code]
	docMu.RUnlock()
	return d, ok
}

// resolveDoc looks up registry docs for this error's code.
func (e *Error) resolveDoc() (DocEntry, bool) {
	if e == nil || e.code == "" {
		return DocEntry{}, false
	}
	return DocFor(e.code)
}
