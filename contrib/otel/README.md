# errx/contrib/otel — OpenTelemetry exception recording for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/ubgo/errx/contrib/otel.svg)](https://pkg.go.dev/github.com/ubgo/errx/contrib/otel)

Bridge [`errx`](https://github.com/ubgo/errx) errors to [OpenTelemetry](https://opentelemetry.io): record the error as an **`exception` span event** using the official `exception.*` semantic conventions, set the span status to `Error`, and (via `Install()`) auto-attach the active **trace and span IDs** to every errx error. Near-universally missing from Go error libraries — here it is one line.

## Contents

- [Why](#why)
- [Install](#install)
- [Step by step](#step-by-step)
- [What gets recorded](#what-gets-recorded)
- [API](#api)
- [Safety](#safety)

## Why

An error in your logs that you can't jump from to the distributed trace is half-useless. This adapter records the error onto the active span (so it shows up in Jaeger/Tempo/Honeycomb/Datadog) **and** stamps the trace/span id onto the error itself so it correlates in logs, Sentry and the wire too — all from the same stable identity.

## Install

```sh
go get github.com/ubgo/errx/contrib/otel
```

Pulls `go.opentelemetry.io/otel` + `.../otel/trace`.

## Step by step

1. **Install the context extractor** (once, at startup) so `Error.Context(ctx)` picks up trace/span ids:

   ```go
   import otelx "github.com/ubgo/errx/contrib/otel"

   otelx.Install()
   ```

2. **Enrich errors from the request context** where you create/wrap them:

   ```go
   func (s *Service) Do(ctx context.Context) error {
       if err := s.step(ctx); err != nil {
           return errx.Wrap(err, "do: step failed").
               WithCode("STEP").
               Context(ctx) // ← attaches trace_id / span_id from the active span
       }
       return nil
   }
   ```

3. **Add the sink to your registry** so reporting also records the exception on the span:

   ```go
   reg := errx.NewRegistry().Add(otelx.NewSink())
   // in middleware, inside the span's context:
   reg.Report(ctx, err)
   ```

## What gets recorded

`(*Sink).Emit` finds the recording span in `ctx` (no-op if none) and:

- adds an `exception` event with attributes:
  - `exception.type` = `domain/code` (or `code`, or `errx.Error`)
  - `exception.message` = operator message
  - `exception.stacktrace` = errx origin frames
  - `error.code`, `error.domain`, `error.fingerprint`
- sets span status to `codes.Error` (skipped for `ClassCancelled` — a client going away is not a server fault).

`Install()` registers an `errx` context extractor returning `SpanContext.TraceID()` / `SpanID()`, so `Error.Context(ctx)` fills `TraceID`/`SpanID` (which then flow into logs, Sentry tags, problem+json `traceId`, etc.).

## API

| Symbol | Purpose |
|---|---|
| `otelx.NewSink() *Sink` | Sink that records the error on the active span. Add to an `errx.Registry`. |
| `otelx.Install()` | Register the trace/span context extractor. Call once at startup. |
| `(*Sink).Name() string` | `"otel"`. |
| `(*Sink).Emit(ctx, *errx.Report) error` | Implements `errx.Sink`. |

## Safety

The sink reads the redacted `errx.Report`; unsafe field values never reach span attributes. Span status is not set for cancelled-class errors to avoid false error rates from client disconnects.

See the [root README](../../README.md) and [`docs/use-cases.md`](../../docs/use-cases.md) for combined Sentry + OTel + Prometheus wiring.
