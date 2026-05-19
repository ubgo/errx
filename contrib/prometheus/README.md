# errx/contrib/prometheus — error metrics for Go by stable identity

[![Go Reference](https://pkg.go.dev/badge/github.com/ubgo/errx/contrib/prometheus.svg)](https://pkg.go.dev/github.com/ubgo/errx/contrib/prometheus) [![Go Report Card](https://goreportcard.com/badge/github.com/ubgo/errx/contrib/prometheus)](https://goreportcard.com/report/github.com/ubgo/errx/contrib/prometheus) [![CI](https://github.com/ubgo/errx/actions/workflows/ci-contrib-prometheus.yml/badge.svg)](https://github.com/ubgo/errx/actions/workflows/ci-contrib-prometheus.yml) [![license](https://img.shields.io/badge/license-Apache--2.0-blue)](../../LICENSE) ![Go](https://img.shields.io/badge/go-1.24%2B-00ADD8?logo=go) [![part of errx](https://img.shields.io/badge/part%20of-errx-6f42c1)](https://github.com/ubgo/errx)

Increment a Prometheus **`errx_errors_total`** counter from [`errx`](https://github.com/ubgo/errx) errors, labelled by the stable identity — `code`, `domain`, `severity`, `class`. Labels come from the machine identity, **not** the variable message, so cardinality stays bounded and the counter groups the same way the fingerprint, Sentry and logs do.

## Contents

- [Why](#why)
- [Install](#install)
- [Step by step](#step-by-step)
- [Metric](#metric)
- [Example queries](#example-queries)
- [API](#api)

## Why

Counting errors by message explodes Prometheus cardinality (every id/count is a new series) and never aligns with how Sentry groups them. `errx` already has a stable `(domain, code)` identity; this adapter uses exactly that for labels, so dashboards, alerts, Sentry issues and logs all pivot on the same key.

## Install

```sh
go get github.com/ubgo/errx/contrib/prometheus
```

Pulls `github.com/prometheus/client_golang`.

## Step by step

1. **Create the sink** with your registry (once, at startup) — it registers the collector:

   ```go
   import (
       "github.com/prometheus/client_golang/prometheus"
       "github.com/prometheus/client_golang/prometheus/promhttp"
       "github.com/ubgo/errx"
       promx "github.com/ubgo/errx/contrib/prometheus"
   )

   reg := errx.NewRegistry().Add(promx.New(prometheus.DefaultRegisterer))
   http.Handle("/metrics", promhttp.Handler())
   ```

2. **Report errors through the registry** (typically in middleware):

   ```go
   if err != nil {
       reg.Report(r.Context(), err) // increments errx_errors_total{...}
   }
   ```

That's it — every reported error bumps the counter with its identity labels.

## Metric

```
errx_errors_total{code="…", domain="…", severity="…", class="…"}  counter
```

`code` and `domain` default to `"unknown"` when the error has none. `severity` is `debug|info|warn|error|fatal`; `class` is `expected|defect|cancelled`.

## Example queries

```promql
# error rate by code over 5m
sum by (code) (rate(errx_errors_total[5m]))

# defects (bugs) only — alert on any
sum(rate(errx_errors_total{class="defect"}[5m])) > 0

# top noisy domains
topk(5, sum by (domain) (increase(errx_errors_total[1h])))
```

## API

| Symbol | Purpose |
|---|---|
| `promx.New(reg prometheus.Registerer) *Sink` | Create + register the `errx_errors_total` `CounterVec`; returns the sink. Pass `nil` to skip registration (advanced/testing). |
| `(*Sink).Name() string` | `"prometheus"`. |
| `(*Sink).Emit(ctx, *errx.Report) error` | Implements `errx.Sink`; increments the counter. |

`New` calls `MustRegister`; a duplicate-collector panic at startup is a programming error worth surfacing loudly.

See the [root README](../../README.md) and [`docs/use-cases.md`](../../docs/use-cases.md) for full observability wiring (Sentry + OTel + Prometheus sharing one fingerprint).
