# errx vs the field

How `errx` compares to the Go error libraries it draws from. `✅` full · `⚠️` partial / manual / external · `❌` absent. `std` = stdlib `errors`, `pkg` = [pkg/errors](https://github.com/pkg/errors), `crdb` = [cockroachdb/errors](https://github.com/cockroachdb/errors), `oops` = [samber/oops](https://github.com/samber/oops), `eris` = [rotisserie/eris](https://github.com/rotisserie/eris), `trace` = [gravitational/trace](https://github.com/gravitational/trace).

The thesis: each existing library covers a slice. The Go ceiling today is `crdb` (~half the surface, heavy + protobuf-coupled, redactable-string ergonomics its loudest criticism) and `oops` (best DX, no wire/observability) — and the two are nearly disjoint in *which* half. `errx` ships the union behind a zero-dependency core with one neutral `Report` seam and no `init()`.

## Identity & classification

| Capability | std | pkg | crdb | oops | eris | trace | **errx** |
|---|:-:|:-:|:-:|:-:|:-:|:-:|:-:|
| `errors.Is/As/Join` drop-in | ✅ | ⚠️ | ✅ | ✅ | ⚠️ | ⚠️ | ✅ |
| Generic `As[T]` | ⚠️ | ❌ | ❌ | ✅ | ❌ | ❌ | ✅ |
| Stable identity `domain`+`code` | ❌ | ❌ | ⚠️ | ⚠️ | ❌ | ❌ | ✅ |
| Semantic category enum | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ |
| Expected / Defect / Cancelled class | ❌ | ❌ | ⚠️ | ❌ | ❌ | ❌ | ✅ |
| Retryable / RetryAfter on the error | ❌ | ❌ | ❌ | ⚠️ | ❌ | ❌ | ✅ |
| Severity / level | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ✅ |

## Context & data

| Capability | std | pkg | crdb | oops | eris | trace | **errx** |
|---|:-:|:-:|:-:|:-:|:-:|:-:|:-:|
| Cause wrapping | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Note without a new wrapper frame | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| Public vs internal message split | ❌ | ❌ | ✅ | ✅ | ❌ | ⚠️ | ✅ |
| Typed structured fields (slog) | ❌ | ❌ | ✅ | ✅ | ⚠️ | ✅ | ✅ |
| Per-field redaction at trust boundary | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ✅ |
| Suppressed (secondary) errors | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ✅ |
| Two-mode accumulation + field path | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| i18n localized messages (BCP-47) | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |

## Diagnostics, wire, observability

| Capability | std | pkg | crdb | oops | eris | trace | **errx** |
|---|:-:|:-:|:-:|:-:|:-:|:-:|:-:|
| Lazy single-alloc stack | ❌ | ❌ | ⚠️ | ⚠️ | ❌ | ⚠️ | ✅ |
| Source-span caret diagnostics | ❌ | ❌ | ❌ | ⚠️ | ❌ | ❌ | ✅ |
| Deterministic fingerprint (not message) | ❌ | ❌ | ⚠️ | ❌ | ❌ | ❌ | ✅ |
| Cross-service serialize + reconstruct | ❌ | ❌ | ✅ | ❌ | ❌ | ⚠️ | ✅ |
| RFC 9457 problem+json | ❌ | ❌ | ❌ | ❌ | ❌ | ⚠️ | ✅ |
| gRPC `Status` + errdetails | ❌ | ❌ | ⚠️ | ❌ | ❌ | ✅ | ✅ |
| ConnectRPC / GraphQL presenters | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| OpenTelemetry exception semconv | ❌ | ❌ | ❌ | ⚠️ | ❌ | ⚠️ | ✅ |
| Sentry (core fingerprint = grouping) | ❌ | ❌ | ✅ | ⚠️ | ⚠️ | ❌ | ✅ |
| Prometheus error metrics | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| Per-fingerprint sampling | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| Error doc registry (code → URL/remediation) | ❌ | ❌ | ⚠️ | ❌ | ❌ | ❌ | ✅ |
| Zero-dependency core | ✅ | ✅ | ❌ | ✅ | ✅ | ⚠️ | ✅ |

## What errx deliberately does NOT do

A best-in-class library knows its edges instead of bolting on a worse version of another tool's job:

- **No retry/circuit-breaker engine.** errx makes the error answer `IsRetryable()`/`RetryAfter()`; a resilience library (`cenkalti/backoff`, etc.) consumes that signal.
- **No `?`-operator / compile-time error-effect typing.** Those are language features Go does not have; errx does not fake them with panics in the core (`result` is an opt-in sub-package, never the primary interface).
- **No logging.** errx produces a structured `Report`; your logger consumes it (`slog.LogValuer` is built in).
- **It does not replace `github.com/ubgo/goerr`.** That thin module stays; `contrib/goerr` is a migration bridge both ways.

## Migration

| From | Path |
|---|---|
| stdlib `fmt.Errorf("%w")` | `errx.Wrap` is a drop-in; `errors.Is/As/Join` keep working |
| `pkg/errors` | replace `errors.Wrap` → `errx.Wrap`; `%+v` still renders a stack |
| `github.com/ubgo/goerr` | `contrib/goerr.From` / `.To` — incremental, both directions |
