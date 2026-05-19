package sentry_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/ubgo/errx"
	errxsentry "github.com/ubgo/errx/contrib/sentry"
)

// capTransport captures events instead of sending them.
type capTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (c *capTransport) Configure(sentry.ClientOptions) {}
func (c *capTransport) SendEvent(e *sentry.Event) {
	c.mu.Lock()
	c.events = append(c.events, e)
	c.mu.Unlock()
}
func (c *capTransport) Flush(time.Duration) bool { return true }
func (c *capTransport) Close()                   {}

func newHub(tr sentry.Transport) *sentry.Hub {
	client, _ := sentry.NewClient(sentry.ClientOptions{Transport: tr, Dsn: ""})
	return sentry.NewHub(client, sentry.NewScope())
}

func TestEmitMapsReport(t *testing.T) {
	tr := &capTransport{}
	hub := newHub(tr)
	sink := errxsentry.New(hub)

	e := errx.New("db exploded").
		WithDomain("billing").WithCode("TX_FAIL").
		WithSeverity(errx.SevWarn).
		With("secret", "leak").WithSafe("order_id", "o-1")

	rep, _ := errx.Snapshot(context.Background(), e)
	if err := sink.Emit(context.Background(), &rep); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	hub.Flush(0)

	if len(tr.events) != 1 {
		t.Fatalf("want 1 event, got %d", len(tr.events))
	}
	ev := tr.events[0]
	if ev.Level != sentry.LevelWarning {
		t.Fatalf("level = %v", ev.Level)
	}
	if ev.Tags["code"] != "TX_FAIL" || ev.Tags["domain"] != "billing" {
		t.Fatalf("tags = %+v", ev.Tags)
	}
	if len(ev.Fingerprint) != 1 || ev.Fingerprint[0] != rep.Fingerprint {
		t.Fatalf("fingerprint = %+v want %q", ev.Fingerprint, rep.Fingerprint)
	}
	ctxFields, _ := ev.Contexts["errx"]
	if ctxFields == nil {
		t.Fatal("errx context missing")
	}
	if ctxFields["secret"] == "leak" {
		t.Fatal("unsafe field leaked to Sentry")
	}
	if len(ev.Exception) != 1 || ev.Exception[0].Mechanism == nil || ev.Exception[0].Mechanism.Handled == nil || !*ev.Exception[0].Mechanism.Handled {
		t.Fatalf("mechanism wrong: %+v", ev.Exception)
	}
}

func TestNameAndNoHubNoPanic(t *testing.T) {
	s := errxsentry.New(nil)
	if s.Name() != "sentry" {
		t.Fatalf("Name = %q", s.Name())
	}
}
