package prometheus_test

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/ubgo/errx"
	errxprom "github.com/ubgo/errx/contrib/prometheus"
)

func TestCounterIncrementsByIdentity(t *testing.T) {
	reg := prometheus.NewRegistry()
	sink := errxprom.New(reg)
	if sink.Name() != "prometheus" {
		t.Fatalf("Name = %q", sink.Name())
	}

	for i := 0; i < 3; i++ {
		rep, _ := errx.Snapshot(context.Background(),
			errx.New("varying message "+string(rune('a'+i))).
				WithDomain("billing").WithCode("TX_FAIL").WithSeverity(errx.SevError))
		_ = sink.Emit(context.Background(), &rep)
	}
	rep, _ := errx.Snapshot(context.Background(),
		errx.New("other").WithDomain("billing").WithCode("NOT_FOUND"))
	_ = sink.Emit(context.Background(), &rep)

	got := testutil.ToFloat64(mustCounter(t, reg, "TX_FAIL"))
	if got != 3 {
		t.Fatalf("TX_FAIL count = %v, want 3 (variable message must not split the series)", got)
	}
}

func mustCounter(t *testing.T, reg *prometheus.Registry, code string) prometheus.Collector {
	t.Helper()
	// Re-create a handle by registering nothing; use a fresh vec query via
	// testutil on the registry's gathered families instead.
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, mf := range mfs {
		if mf.GetName() == "errx_errors_total" {
			for _, m := range mf.GetMetric() {
				for _, l := range m.GetLabel() {
					if l.GetName() == "code" && l.GetValue() == code {
						c := prometheus.NewCounter(prometheus.CounterOpts{Name: "x"})
						c.Add(m.GetCounter().GetValue())
						return c
					}
				}
			}
		}
	}
	t.Fatalf("counter for code %q not found", code)
	return nil
}
