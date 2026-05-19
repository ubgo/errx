# errx/contrib/sentry — Sentry error reporting for Go with stable grouping

[![Go Reference](https://pkg.go.dev/badge/github.com/ubgo/errx/contrib/sentry.svg)](https://pkg.go.dev/github.com/ubgo/errx/contrib/sentry) [![Go Report Card](https://goreportcard.com/badge/github.com/ubgo/errx/contrib/sentry)](https://goreportcard.com/report/github.com/ubgo/errx/contrib/sentry) [![CI](https://github.com/ubgo/errx/actions/workflows/ci-contrib-sentry.yml/badge.svg)](https://github.com/ubgo/errx/actions/workflows/ci-contrib-sentry.yml) [![license](https://img.shields.io/badge/license-Apache--2.0-blue)](../../LICENSE) ![Go](https://img.shields.io/badge/go-1.24%2B-00ADD8?logo=go) [![part of errx](https://img.shields.io/badge/part%20of-errx-6f42c1)](https://github.com/ubgo/errx)

Report [`errx`](https://github.com/ubgo/errx) errors to [Sentry](https://sentry.io) with the **core deterministic fingerprint as the Sentry grouping key** — so the same logical bug groups identically in Sentry, Prometheus, logs and on the wire, instead of relying on Sentry's server-side guessing. Severity → level, `code`/`domain` → tags, safe fields → context, `mechanism.handled = true`, oldest-first stack frames.

## Contents

- [Why](#why)
- [Install](#install)
- [Step by step](#step-by-step)
- [What gets sent](#what-gets-sent)
- [Sampling](#sampling)
- [API](#api)
- [Safety](#safety)

## Why

Sentry groups by a guessed fingerprint, so the *same* bug often fragments into many issues when the message contains variable data (ids, counts). `errx` already computes a stable, message-independent fingerprint; this adapter hands it to Sentry as the explicit grouping key. One bug → one Sentry issue, and the same id everywhere else.

## Install

```sh
go get github.com/ubgo/errx/contrib/sentry
```

Pulls `github.com/getsentry/sentry-go`.

## Step by step

1. **Initialise Sentry** as usual (in `main`):

   ```go
   sentry.Init(sentry.ClientOptions{Dsn: os.Getenv("SENTRY_DSN")})
   defer sentry.Flush(2 * time.Second)
   ```

2. **Build an errx registry with the Sentry sink** (once, at startup):

   ```go
   import (
       "github.com/getsentry/sentry-go"
       "github.com/ubgo/errx"
       errxsentry "github.com/ubgo/errx/contrib/sentry"
   )

   reg := errx.NewRegistry().
       Add(errxsentry.New(nil)) // nil → resolve hub from ctx, else sentry.CurrentHub()
   ```

   Pass an explicit `*sentry.Hub` instead of `nil` to pin one.

3. **Report from your error middleware**:

   ```go
   if err != nil {
       reg.Report(r.Context(), err) // projects + redacts + sends to Sentry
       httpx.Write(w, r, err)
   }
   ```

## What gets sent

| Sentry field | From |
|---|---|
| `level` | `errx` severity (`SevWarn → warning`, `SevFatal → fatal`, …) |
| `message` | operator message |
| `fingerprint` | **the core `errx.Fingerprint()`** (explicit grouping) |
| `tags.code` / `tags.domain` / `tags.class` / `tags.trace_id` | identity |
| `contexts.errx` | safe fields + `hint` + `owner` |
| `exception[0].type` | `domain/code` (or `code`, or `errx.Error`) |
| `exception[0].mechanism` | `{type:"errx", handled:true}` |
| `exception[0].stacktrace` | errx origin frames, oldest-first (Sentry order) |

## Sampling

Wrap the sink so one hot error can't blow your Sentry quota:

```go
reg := errx.NewRegistry().Add(
    errx.NewSampledSink(errxsentry.New(nil), errx.PerFingerprint(time.Minute, 5)),
)
```

At most 5 events per fingerprint per minute reach Sentry; the next allowed event carries an `errx.sampled_dropped` count so suppression is visible, not silent.

## API

| Symbol | Purpose |
|---|---|
| `errxsentry.New(hub *sentry.Hub) *Sink` | Create the sink. `nil` resolves the hub per-call from ctx → `CurrentHub()`. |
| `(*Sink).Name() string` | `"sentry"`. |
| `(*Sink).Emit(ctx, *errx.Report) error` | Implements `errx.Sink`; called by the registry. |

## Safety

The sink consumes the already-redacted `errx.Report`: unsafe field values are replaced with a redaction marker before they reach Sentry. The operator message is sent (Sentry is an operator tool); the public message is not conflated with it.

See the [root README](../../README.md) for registry wiring and [`docs/use-cases.md`](../../docs/use-cases.md) for the full observability setup.
