package errx

import (
	"context"
	"sync"
	"testing"
	"time"
)

type countSink struct {
	mu   sync.Mutex
	n    int
	last *Report
}

func (c *countSink) Name() string { return "count" }
func (c *countSink) Emit(_ context.Context, r *Report) error {
	c.mu.Lock()
	c.n++
	c.last = r
	c.mu.Unlock()
	return nil
}

func TestSampledSinkRateLimitsPerFingerprint(t *testing.T) {
	base := time.Unix(0, 0)
	cur := base
	now = func() time.Time { return cur }
	defer func() { now = time.Now }()

	cs := &countSink{}
	s := NewSampledSink(cs, PerFingerprint(time.Minute, 3))

	r := &Report{Fingerprint: "fp-A"}
	for i := 0; i < 100; i++ {
		_ = s.Emit(context.Background(), r)
	}
	if cs.n != 3 {
		t.Fatalf("burst not enforced: forwarded %d, want 3", cs.n)
	}

	// A different fingerprint has its own budget.
	_ = s.Emit(context.Background(), &Report{Fingerprint: "fp-B"})
	if cs.n != 4 {
		t.Fatalf("independent fingerprint blocked: %d", cs.n)
	}

	// Next window: allowed again, and the drop count is surfaced.
	cur = base.Add(time.Minute + time.Second)
	_ = s.Emit(context.Background(), r)
	if cs.n != 5 {
		t.Fatalf("window did not reset: %d", cs.n)
	}
	var sawDropped bool
	for _, f := range cs.last.Fields {
		if f.Key == "errx.sampled_dropped" {
			sawDropped = true
		}
	}
	if !sawDropped {
		t.Fatal("dropped count not surfaced on the next allowed report")
	}
}

func TestSampledSinkPassThroughWhenDisabledOrNoFingerprint(t *testing.T) {
	cs := &countSink{}
	s := NewSampledSink(cs, PerFingerprint(0, 0)) // disabled
	for i := 0; i < 10; i++ {
		_ = s.Emit(context.Background(), &Report{Fingerprint: "x"})
	}
	if cs.n != 10 {
		t.Fatalf("disabled sampling should pass all, got %d", cs.n)
	}

	cs2 := &countSink{}
	s2 := NewSampledSink(cs2, PerFingerprint(time.Minute, 1))
	for i := 0; i < 5; i++ {
		_ = s2.Emit(context.Background(), &Report{}) // no fingerprint
	}
	if cs2.n != 5 {
		t.Fatalf("no-fingerprint reports must always pass, got %d", cs2.n)
	}
	if s2.Name() != "count" {
		t.Fatalf("Name should delegate, got %q", s2.Name())
	}
}
