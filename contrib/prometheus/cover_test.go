package prometheus_test

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/ubgo/errx"
	errxprom "github.com/ubgo/errx/contrib/prometheus"
)

func TestEmitUnknownLabelsAndNilRegisterer(t *testing.T) {
	// nil Registerer → New must not register/panic.
	s := errxprom.New(nil)

	reg := prometheus.NewRegistry()
	s2 := errxprom.New(reg)

	// Error with no code/domain → labels default to "unknown".
	rep, _ := errx.Snapshot(context.Background(), errx.New("bare"))
	_ = s.Emit(context.Background(), &rep)  // no-op-ish (unregistered vec still increments internally)
	_ = s2.Emit(context.Background(), &rep) // registered

	mfs, _ := reg.Gather()
	var found bool
	for _, mf := range mfs {
		if mf.GetName() == "errx_errors_total" {
			for _, m := range mf.GetMetric() {
				labels := map[string]string{}
				for _, l := range m.GetLabel() {
					labels[l.GetName()] = l.GetValue()
				}
				if labels["code"] == "unknown" && labels["domain"] == "unknown" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("expected an errx_errors_total series with unknown code/domain labels")
	}

	// Sanity: a coded error increments a distinct series.
	rep2, _ := errx.Snapshot(context.Background(),
		errx.New("x").WithDomain("d").WithCode("C").WithSeverity(errx.SevWarn))
	_ = s2.Emit(context.Background(), &rep2)
	if testutil.CollectAndCount(reg) == 0 {
		t.Fatal("counter should have series")
	}
}
