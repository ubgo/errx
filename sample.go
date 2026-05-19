package errx

import (
	"context"
	"sync"
	"time"
)

// SampledSink wraps a Sink with per-fingerprint rate limiting: when one
// logical error fires thousands of times a minute it forwards at most
// `burst` reports per `window` for that fingerprint and drops the rest, so
// a single hot error cannot blow the Sentry bill or bury rare errors.
// Different fingerprints are limited independently. Reports with no
// fingerprint are always forwarded.
//
//	reg := errx.NewRegistry().Add(
//		errx.SampledSink(sentry.New(client), errx.PerFingerprint(time.Minute, 5)))
type SampledSink struct {
	inner  Sink
	window time.Duration
	burst  int

	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	windowStart time.Time
	count       int
	dropped     int
}

// SamplePolicy configures a SampledSink.
type SamplePolicy struct {
	Window time.Duration
	Burst  int
}

// PerFingerprint builds a policy allowing burst reports per window per
// distinct fingerprint.
func PerFingerprint(window time.Duration, burst int) SamplePolicy {
	return SamplePolicy{Window: window, Burst: burst}
}

// NewSampledSink wraps inner with the given policy. A non-positive burst or
// window disables sampling (everything passes through).
func NewSampledSink(inner Sink, p SamplePolicy) *SampledSink {
	return &SampledSink{
		inner:   inner,
		window:  p.Window,
		burst:   p.Burst,
		buckets: make(map[string]*bucket),
	}
}

// Name returns the inner sink's name.
func (s *SampledSink) Name() string { return s.inner.Name() }

// now is overridable for tests.
var now = time.Now

// allow reports whether a report with the given fingerprint may pass, and
// how many had been dropped since the last allowed one in this window.
func (s *SampledSink) allow(fp string) (bool, int) {
	if s.burst <= 0 || s.window <= 0 || fp == "" {
		return true, 0
	}
	t := now()
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.buckets[fp]
	if b == nil || t.Sub(b.windowStart) >= s.window {
		prevDropped := 0
		if b != nil {
			prevDropped = b.dropped
		}
		s.buckets[fp] = &bucket{windowStart: t, count: 1}
		return true, prevDropped
	}
	if b.count < s.burst {
		b.count++
		return true, 0
	}
	b.dropped++
	return false, 0
}

// Emit forwards the report to the inner sink unless this fingerprint is
// over its budget for the current window. When a new window opens it
// attaches a "dropped" field recording how many were suppressed, so the
// suppression is visible rather than silent.
func (s *SampledSink) Emit(ctx context.Context, r *Report) error {
	ok, dropped := s.allow(r.Fingerprint)
	if !ok {
		return nil
	}
	if dropped > 0 {
		cp := *r
		cp.Fields = append(append([]Field(nil), r.Fields...),
			Field{Key: "errx.sampled_dropped", Value: dropped, Safe: true})
		return s.inner.Emit(ctx, &cp)
	}
	return s.inner.Emit(ctx, r)
}
