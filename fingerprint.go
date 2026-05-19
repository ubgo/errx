package errx

import (
	"hash/fnv"
	"strconv"
)

// Fingerprint returns a deterministic, message-independent grouping key:
// FNV-1a of domain + code + the origin frame (function:line). It is stable
// across instances and processes, so the SAME bug groups identically in
// Sentry, the Prometheus label, the log field, and the wire — solving the
// "four disconnected grouping mechanisms" problem. High-cardinality data in
// the message never perturbs it.
func (e *Error) Fingerprint() string {
	if e == nil {
		return ""
	}
	if e.fp != "" {
		return e.fp
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(e.domain))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(e.code))
	_, _ = h.Write([]byte{0})
	of := e.originFrame()
	_, _ = h.Write([]byte(of.Function))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strconv.Itoa(of.Line)))
	e.fp = strconv.FormatUint(h.Sum64(), 16)
	return e.fp
}

// Fingerprint returns the grouping key of the nearest *Error in err, or "".
func Fingerprint(err error) string {
	if e := Get(err); e != nil {
		return e.Fingerprint()
	}
	return ""
}
