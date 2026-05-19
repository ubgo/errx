// Package prometheus increments an errors_total counter from errx Reports,
// labelled by code/domain/severity/class. The label values come from the
// stable identity (not the variable message), so cardinality stays bounded
// and the counter groups the same way the fingerprint, Sentry and logs do.
package prometheus

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/ubgo/errx"
)

// Sink is an errx.Sink that counts errors.
type Sink struct {
	counter *prometheus.CounterVec
}

// New creates the errors_total CounterVec, registers it with reg, and
// returns a sink. Panics only if reg already has a colliding collector
// (a programming error worth surfacing loudly at startup).
func New(reg prometheus.Registerer) *Sink {
	c := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "errx_errors_total",
		Help: "Total errx errors by stable identity.",
	}, []string{"code", "domain", "severity", "class"})
	if reg != nil {
		reg.MustRegister(c)
	}
	return &Sink{counter: c}
}

// Name identifies the sink.
func (*Sink) Name() string { return "prometheus" }

// Emit increments the counter for this report's identity.
func (s *Sink) Emit(_ context.Context, r *errx.Report) error {
	code := r.Code
	if code == "" {
		code = "unknown"
	}
	domain := r.Domain
	if domain == "" {
		domain = "unknown"
	}
	s.counter.WithLabelValues(code, domain, r.Severity.String(), r.Class.String()).Inc()
	return nil
}
