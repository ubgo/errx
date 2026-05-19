package sentry_test

import (
	"context"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/ubgo/errx"
	errxsentry "github.com/ubgo/errx/contrib/sentry"
)

func TestLevelMappingAllSeverities(t *testing.T) {
	cases := []struct {
		sev   errx.Severity
		level sentry.Level
	}{
		{errx.SevDebug, sentry.LevelDebug},
		{errx.SevInfo, sentry.LevelInfo},
		{errx.SevWarn, sentry.LevelWarning},
		{errx.SevError, sentry.LevelError},
		{errx.SevFatal, sentry.LevelFatal},
	}
	for _, c := range cases {
		tr := &capTransport{}
		hub := newHub(tr)
		rep, _ := errx.Snapshot(context.Background(),
			errx.New("m").WithCode("C").WithSeverity(c.sev))
		_ = errxsentry.New(hub).Emit(context.Background(), &rep)
		hub.Flush(0)
		if len(tr.events) != 1 || tr.events[0].Level != c.level {
			t.Fatalf("severity %v → level %v, want %v", c.sev, tr.events[0].Level, c.level)
		}
	}
}

func TestExceptionTypeVariants(t *testing.T) {
	cases := []struct {
		domain, code, want string
	}{
		{"billing", "TX", "billing/TX"},
		{"", "TX", "TX"},
		{"", "", "errx.Error"},
	}
	for _, c := range cases {
		tr := &capTransport{}
		hub := newHub(tr)
		e := errx.New("m")
		if c.domain != "" {
			e = e.WithDomain(c.domain)
		}
		if c.code != "" {
			e = e.WithCode(c.code)
		}
		rep, _ := errx.Snapshot(context.Background(), e)
		_ = errxsentry.New(hub).Emit(context.Background(), &rep)
		hub.Flush(0)
		if got := tr.events[0].Exception[0].Type; got != c.want {
			t.Fatalf("domain=%q code=%q → type %q, want %q", c.domain, c.code, got, c.want)
		}
	}
}

func TestHubFromContext(t *testing.T) {
	tr := &capTransport{}
	hub := newHub(tr)
	ctx := sentry.SetHubOnContext(context.Background(), hub)

	// New(nil) → resolve hub from context.
	rep, _ := errx.Snapshot(ctx, errx.New("m").WithCode("C"))
	_ = errxsentry.New(nil).Emit(ctx, &rep)
	hub.Flush(0)
	if len(tr.events) != 1 {
		t.Fatalf("hub-from-context path: want 1 event, got %d", len(tr.events))
	}
}

func TestEmitNoClientHubIsSafe(t *testing.T) {
	// New(nil) with a bare context falls back to CurrentHub(); with no
	// client bound it must not panic and must be a no-op-ish call.
	rep, _ := errx.Snapshot(context.Background(), errx.New("m").WithCode("C"))
	if err := errxsentry.New(nil).Emit(context.Background(), &rep); err != nil {
		t.Fatalf("Emit with default hub should be safe: %v", err)
	}
}
