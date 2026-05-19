# errx

[![Go Reference](https://pkg.go.dev/badge/github.com/ubgo/errx.svg)](https://pkg.go.dev/github.com/ubgo/errx)
[![Go Report Card](https://goreportcard.com/badge/github.com/ubgo/errx)](https://goreportcard.com/report/github.com/ubgo/errx)
[![CI](https://github.com/ubgo/errx/actions/workflows/ci.yml/badge.svg)](https://github.com/ubgo/errx/actions/workflows/ci.yml)
[![Go 1.24+](https://img.shields.io/badge/go-1.24%2B-00ADD8)](https://go.dev/dl/)
[![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue)](./LICENSE)

One coherent structured error for Go. A single value carries:

- a **stable machine identity** — `domain` + `code` — separate from the human message
- an **expected / defect / cancelled class** so middleware and alerting can branch correctly
- a **separate operator and end-user message** (internal detail never leaks to clients)
- **typed structured fields** with **per-field redaction** (safe-by-default at a trust boundary)
- **retry and severity metadata** the resilience/log layers read off the error itself
- a **lazily captured stack** (single `runtime.Callers`, symbolized only when printed)
- **suppressed (secondary) errors** so a failing `Close()` never masks the real failure
- a **deterministic, message-independent fingerprint** — the same bug groups identically in Sentry, Prometheus, logs, and on the wire

…while staying a drop-in over the standard library's `errors.Is` / `errors.As` / `errors.Join`.

The core imports **nothing beyond the standard library**. Observability and transport integrations (Sentry, OpenTelemetry, RFC 9457, gRPC, GraphQL, Prometheus, …) live under `contrib/*` as separate modules that consume one neutral `Report` via `Snapshot` — the core never imports them, and there is **no `init()` magic**: registration is explicit.

## Install

```sh
go get github.com/ubgo/errx
```

Requires Go 1.24+.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/ubgo/errx"
)

func main() {
	err := errx.New("db: deadlock detected on user_orders").
		WithDomain("billing").
		WithCode("TX_RETRY").
		WithPublic("Something went wrong, please retry").
		WithHint("retry with exponential backoff").
		WithRetryable(0).
		With("table", "user_orders"). // unsafe: redacted at a trust boundary
		WithSafe("shard", "eu-1")     // safe: never redacted

	fmt.Println(err.Error())                 // operator message
	fmt.Println(err.Public("Unknown error")) // end-user-safe message
	fmt.Println(errx.Code(err))              // TX_RETRY
	fmt.Println(errx.IsRetryable(err))       // true
	fmt.Println(err.Fingerprint())           // stable grouping key
}
```

## Wrapping

`Wrap` is **transparent** by default — `errors.Is`/`errors.As` traverse the cause, and an inner `errx.Error`'s identity/class is inherited so an outer annotation doesn't erase precise classification:

```go
if err != nil {
	return errx.Wrap(err, "load orders").WithCode("DB_READ")
}
```

Call `.Opaque()` to make a deliberate barrier that hides a dependency's sentinels from your package's callers.

`Note` attaches a breadcrumb **without** adding a wrapper frame — the answer to `fmt.Errorf("…: %w")` spam:

```go
errx.Note(err, "attempt", 3)
```

## Accumulating errors

Collect every failure instead of stopping at the first (the validation gap in Go):

```go
acc := errx.NewAccumulator()
acc.Add("name", validateName(in.Name))
acc.Add("email", validateEmail(in.Email))
return acc.ErrorOrNil() // nil, or one error listing every failure with its path
```

`errx.Accumulate(fns…)` and `errx.ParAccumulate(fns…)` (concurrent) are the functional forms.

## Structured logging

`*errx.Error` implements `slog.LogValuer`, so it self-describes to `log/slog` as a structured group (code, domain, class, severity, fingerprint, safe fields) — unsafe fields redacted automatically:

```go
slog.Error("request failed", "err", err)
```

## Reporting (the adapter seam)

Every integration consumes one neutral `Report`. Wire sinks explicitly:

```go
reg := errx.NewRegistry().
	Add(sentryadapter.New(client)).
	Add(oteladapter.New())

reg.Report(ctx, err) // projects + fans out; unsafe fields already redacted
```

Sampling is a sink decorator — one line keeps a hot error from blowing the bill:

```go
reg.Add(errx.NewSampledSink(sentryadapter.New(client), errx.PerFingerprint(time.Minute, 5)))
```

### Adapters (each a separate, opt-in module)

| Module | Purpose |
|---|---|
| `contrib/httpx` | RFC 9457 `application/problem+json` (stdlib-only) |
| `contrib/grpc` | `google.rpc.Status` + ErrorInfo/RetryInfo/LocalizedMessage/Help |
| `contrib/connect` | ConnectRPC error + ErrorInfo detail |
| `contrib/graphql` | gqlgen `ErrorPresenter` (extensions.code) |
| `contrib/sentry` | Capture with core fingerprint as the grouping key |
| `contrib/otel` | exception span event + semconv + ctx extractor |
| `contrib/prometheus` | `errx_errors_total{code,domain,severity,class}` |
| `contrib/goerr` | migration shim to/from `github.com/ubgo/goerr` |

The `codec` in core (`Encode`/`Decode`) round-trips an error across a service boundary preserving identity, class, retry metadata and fingerprint — unsafe fields are never serialized.

### Compiler-grade diagnostics

Attach source and labels and render a miette-style report:

```go
err := errx.New(`unknown column "emial"`).
	WithCode("PG_UNDEFINED_COLUMN").
	WithHint(`did you mean "email"?`).
	WithURL("https://errors.example.com/PG_UNDEFINED_COLUMN").
	WithSource("query.sql", sql).
	WithLabel(off, len("emial"), "no such column")

fmt.Println(diag.String(err)) // or diag.Fprint(os.Stderr, err, diag.Auto())
```

```
error[PG_UNDEFINED_COLUMN]: unknown column "emial"
  --> query.sql
3 | WHERE  emial = $1
  |        ^^^^^ no such column
help: did you mean "email"?
docs: https://errors.example.com/PG_UNDEFINED_COLUMN
```

`diag.Auto()` switches to a prose "narratable" form under `NO_COLOR`/`CI`/non-TTY (screen-reader friendly).

### Less boilerplate — `errxgen`

Go has no derive macros. `cmd/errxgen` closes the gap: annotate a struct and `go generate` emits the boilerplate (generated code imports only `fmt`):

```go
//go:generate go run github.com/ubgo/errx/cmd/errxgen .

//errxgen: message="open %s: permission denied (uid %d)" args=Path,UID code=IO_DENIED unwrap=Err
type DeniedError struct {
	Path string
	UID  int
	Err  error
}
```

`errx.Code`/`HasCode` also honor any error exposing `Code() string`, so generated and foreign typed errors are first-class without being `*errx.Error`.

`Error.Trace()` returns the return-trace (one frame per wrap layer — *where the error traveled*, distinct from the origin stack), also shown in `%+v`.

The optional `result` subpackage offers a generics `Result[T]` railway layer for code that prefers it — never required.

## Documentation

- [`docs/design.md`](./docs/design.md) — the `Report` seam, zero-dep core, no-`init()`, wrapping/class/stack rationale
- [`docs/comparison.md`](./docs/comparison.md) — feature matrix vs stdlib / pkg.errors / cockroachdb / oops / eris / gravitational, and explicit non-goals
- [`docs/use-cases.md`](./docs/use-cases.md) — copy-pasteable patterns (HTTP, gRPC, validation, retry, cross-service, observability, diagnostics)

## Status

Pre-1.0. Core + `diag` + `result` + 8 `contrib/*` adapters implemented and tested under `-race` (11 packages, ~120 tests, core ≈89% coverage). API may change before `v1.0.0`. See [`CHANGELOG.md`](./CHANGELOG.md).

## License

Apache-2.0. See [`LICENSE`](./LICENSE) and [`NOTICE`](./NOTICE).
