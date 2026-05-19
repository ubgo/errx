# errx — the structured error library for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/ubgo/errx.svg)](https://pkg.go.dev/github.com/ubgo/errx) [![Go Report Card](https://goreportcard.com/badge/github.com/ubgo/errx)](https://goreportcard.com/report/github.com/ubgo/errx) [![CI](https://github.com/ubgo/errx/actions/workflows/ci.yml/badge.svg)](https://github.com/ubgo/errx/actions/workflows/ci.yml) [![codecov](https://img.shields.io/badge/coverage-94%25-brightgreen)](https://github.com/ubgo/errx/actions/workflows/ci.yml) [![tag](https://img.shields.io/github/v/tag/ubgo/errx?sort=semver&label=release)](https://github.com/ubgo/errx/tags) [![license](https://img.shields.io/badge/license-Apache--2.0-blue)](./LICENSE) ![Go](https://img.shields.io/badge/go-1.24%2B-00ADD8?logo=go) [![PRs welcome](https://img.shields.io/badge/PRs-welcome-brightgreen)](./CONTRIBUTING.md) [![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-fe5196?logo=conventionalcommits)](https://www.conventionalcommits.org)

**errx** is a complete, production-grade **error handling library for Go (Golang)**. One coherent error value carries a stable machine identity, a separate operator and end-user message, typed and redactable structured fields, retry and severity metadata, a lazily-captured stack plus a return-trace, suppressed errors, a deterministic fingerprint, and compiler-grade diagnostics — while staying a **drop-in over the standard library `errors.Is` / `errors.As` / `errors.Join`**.

It is the error package that replaces the usual pile of five: `pkg/errors` for stacks, `go-multierror` for aggregation, a hand-rolled struct for codes, a bespoke HTTP/gRPC mapper, and per-vendor Sentry/OpenTelemetry glue. The **core has zero third-party dependencies**; every observability and transport integration (Sentry, OpenTelemetry, RFC 9457 problem+json, gRPC, Connect, GraphQL, Prometheus) is a separate, opt-in `contrib/*` module that consumes one neutral `Report` — with **no `init()` magic**.

```go
err := errx.New("db: deadlock on user_orders").
    WithDomain("billing").WithCode("TX_RETRY").
    WithPublic("Something went wrong, please retry").
    WithRetryable(0).
    WithSafe("shard", "eu-1")          // safe field — survives to logs/wire
    // .With("token", secret)          // unsafe field — auto-redacted at any boundary

httpx.Write(w, r, err)                  // → 503 application/problem+json
slog.Error("request failed", "err", err) // structured, redacted, fingerprinted
```

## Table of contents

- [Why errx](#why-errx)
- [Install](#install)
- [Quick start](#quick-start)
- [Core concepts](#core-concepts)
- [Full API tour](#full-api-tour)
- [Integrations (contrib)](#integrations-contrib)
- [Observability wiring](#observability-wiring)
- [Diagnostics (miette-style)](#diagnostics-miette-style)
- [Code generation — errxgen](#code-generation--errxgen)
- [Optional Result\[T\] layer](#optional-resultt-layer)
- [Testing helpers](#testing-helpers)
- [FAQ](#faq)
- [Documentation](#documentation)

## Why errx

Idiomatic Go error handling loses information at every boundary. A database error becomes a string, the string becomes an HTTP 500, the 500 tells the user nothing and the on-call engineer even less. Existing libraries each solve a slice — `cockroachdb/errors` has wire portability and redaction but is heavy and protobuf-coupled; `samber/oops` has great DX but no OpenTelemetry, gRPC, or wire format. **No Go library ships the coherent union.** errx does:

- **Stable identity** — `domain` + `code`, separate from the human message, so dashboards and clients branch on it even when wording changes.
- **Expected / Defect / Cancelled class** — middleware returns 4xx + retry, alerts on bugs, stays quiet on client-cancelled.
- **Operator vs end-user message** — internal detail never leaks to clients.
- **Typed structured fields with per-field redaction** — safe-by-default at any trust boundary; no redactable-string tax.
- **Retry & severity on the error itself** — resilience and logging layers read it, not a type switch.
- **Lazy single-alloc stack + return-trace** — ~15 ns capture, symbolized only when printed; plus *where the error traveled*.
- **Suppressed errors** — a failing `defer Close()` never masks the real failure (a classic Go bug Go has no stdlib fix for).
- **Two-mode accumulation** — collect *every* validation failure with field paths, not just the first.
- **Deterministic fingerprint** — the same bug groups identically in Sentry, Prometheus, logs, and on the wire.
- **Cross-service JSON codec** — serialize an error, reconstruct the typed error on the other side; identity survives, unsafe fields never serialize.
- **Compiler-grade diagnostics** — source spans with caret underlines, error codes, doc URLs, dual graphical/narratable renderers.
- **i18n** — BCP-47 localized end-user messages with language fallback.
- **Zero-dependency core, explicit adapters, no `init()`.**

See [`docs/comparison.md`](./docs/comparison.md) for the feature-by-feature matrix versus stdlib, pkg/errors, cockroachdb/errors, samber/oops, eris and gravitational/trace, and the explicit non-goals.

## Install

Requires **Go 1.24+**.

```sh
go get github.com/ubgo/errx
```

Each integration is a **separate module** — install only what you use:

```sh
go get github.com/ubgo/errx/contrib/httpx        # RFC 9457 problem+json (zero extra deps)
go get github.com/ubgo/errx/contrib/grpc         # google.rpc.Status + error details
go get github.com/ubgo/errx/contrib/connect      # ConnectRPC
go get github.com/ubgo/errx/contrib/graphql      # gqlgen error presenter
go get github.com/ubgo/errx/contrib/sentry       # Sentry capture
go get github.com/ubgo/errx/contrib/otel         # OpenTelemetry
go get github.com/ubgo/errx/contrib/prometheus   # error metrics
go get github.com/ubgo/errx/contrib/goerr        # migration from github.com/ubgo/goerr
```

The core imports **nothing beyond the Go standard library**. You only pull a third-party dependency when you import the matching `contrib` module.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/ubgo/errx"
)

func loadOrder(id string) error {
	// ... db call fails ...
	return errx.New("db: no rows for order " + id).
		WithDomain("orders").
		WithCode("NOT_FOUND").
		WithPublic("Order not found").
		WithHint("verify the order id").
		WithSafe("order_id", id)
}

func main() {
	err := loadOrder("o-42")

	fmt.Println(err.Error())                    // operator message (logs)
	fmt.Println(errx.Get(err).Public("error"))  // end-user-safe message
	fmt.Println(errx.Code(err))                 // NOT_FOUND
	fmt.Println(errx.Get(err).Fingerprint())    // stable grouping key
}
```

## Core concepts

### Construction and wrapping

| Function | Purpose |
|---|---|
| `errx.New(msg)` / `errx.Newf(fmt, …)` | Create a leaf error with an origin stack. |
| `errx.Wrap(err, msg)` / `errx.Wrapf(err, fmt, …)` | Annotate a cause. **Transparent** by default (`errors.Is/As` traverse); inner identity/class inherited. Returns `nil` if `err` is `nil`. |
| `errx.Note(err, key, value)` | Attach a breadcrumb **without** a new wrapper frame — the cure for `fmt.Errorf("…: %w")` spam. |
| `errx.Join(errs…)` | Bundle errors; `errors.Is/As` match any member. |
| `.Opaque()` | Make the error a barrier so `errors.Is/As` do **not** see the cause (deliberately hide a dependency's sentinels). |

### Identity, class, severity

```go
errx.New("…").
    WithDomain("billing").          // namespace
    WithCode("CARD_DECLINED").      // stable machine code
    WithClass(errx.ClassExpected).  // ClassExpected | ClassDefect | ClassCancelled
    WithSeverity(errx.SevWarn).     // SevDebug | SevInfo | SevWarn | SevError | SevFatal
    WithRetryable(3 * time.Second). // retryable + suggested delay
    WithOwner("payments")           // routing/alerting
```

Read them back from anywhere in a chain: `errx.Code(err)`, `errx.HasCode(err, "X")`, `errx.FindByCode(err, "X")`, `errx.ClassOf(err)`, `errx.IsExpected/IsDefect/IsCancelled(err)`, `errx.IsRetryable(err)`, `errx.RetryAfter(err)`, `errx.Fingerprint(err)`. `errx.Code` / `HasCode` also recognise any error exposing a `Code() string` method, so foreign and generated typed errors are first-class.

### Messages: operator vs end-user vs localized

```go
errx.New("pq: deadlock detected on user_orders"). // operator (logs, .Error())
    WithPublic("Something went wrong, please retry"). // end-user-safe
    WithLocalized("fr-FR", "Une erreur est survenue"). // i18n (BCP-47)
    WithLocalized("ja", "エラーが発生しました")

e := errx.Get(err)
e.Public("fallback")          // end-user message, never the operator detail
e.Localized("fr-CA", "fb")    // → French message via language fallback (fr-CA → fr-FR)
```

### Structured fields and redaction (safe-by-default)

```go
errx.New("charge failed").
    WithSafe("order_id", "o-1").  // safe → appears in logs, problem+json, Sentry, wire
    With("card_pan", pan)         // UNSAFE → replaced with ‹redacted› at every boundary
```

Redaction is enforced **once**, in `Snapshot`, before any sink or wire encoder sees the data — safe by construction, with none of the redactable-string ergonomic cost.

### Suppressed errors and error-path cleanup

```go
func process(path string) (err error) {
	f, err := os.Open(path)
	if err != nil {
		return errx.Wrap(err, "open input").WithCode("IO")
	}
	defer errx.CloseSuppressing(&err, f)            // close error becomes secondary, never masks
	defer errx.OnError(&err, func() { rollback() }) // runs only on the failure path (Zig errdefer)
	return parse(f)
}
```

`errx.OnSuccess`, `errx.AppendSuppressed` are also available.

### Accumulation — collect every failure

```go
acc := errx.NewAccumulator()
acc.Add("name", validateName(in.Name))
acc.Add("email", validateEmail(in.Email))
acc.AddErr(checkQuota(ctx))
return acc.ErrorOrNil() // nil, or one VALIDATION error listing every failure with its field path
```

`errx.Accumulate(fns…)` and the concurrent `errx.ParAccumulate(fns…)` are the functional forms.

### Panic recovery

```go
func handler() (err error) {
	defer errx.Recover(&err) // panic → ClassDefect/SevFatal error, original preserved as cause
	return risky()
}
// errx.RecoverDo(fn) and errx.Recovered(err) are also provided.
```

### Cross-service wire codec

```go
blob, _ := errx.Encode(err)   // JSON; unsafe fields are NOT serialized
got, _  := errx.Decode(blob)  // typed *errx.Error rebuilt
errx.HasCode(got, "QUOTA")    // identity survived the hop
got.Fingerprint() == err...   // same fingerprint → groups together everywhere
errx.RegisterCodeMigration("OLD_CODE", "NEW_CODE") // cross-version skew tolerance
```

## Full API tour

| Area | Symbols |
|---|---|
| Construct | `New` `Newf` `Wrap` `Wrapf` `Note` `Join` |
| Identity | `WithDomain` `WithCode` `WithClass` `WithSeverity` `WithRetryable` `WithOwner` |
| Messages | `WithPublic` `WithHint` `WithLocalized` · `Public` `Remediation` `Localized` `LocaleMessages` |
| Fields | `With` `WithSafe` · `Fields` |
| Inspect | `Get` `Code` `HasCode` `FindByCode` `As[T]` `ClassOf` `IsExpected` `IsDefect` `IsCancelled` `IsRetryable` `RetryAfter` `Fingerprint` |
| Stack | `Frames` (origin) · `Trace` (return-trace) |
| Suppress / cleanup | `Suppress` `OnError` `OnSuccess` `CloseSuppressing` `AppendSuppressed` |
| Accumulate | `NewAccumulator` `Accumulate` `ParAccumulate` |
| Panic | `Recover` `RecoverDo` `Recovered` |
| Context | `RegisterContextExtractor` · `Error.Context(ctx)` |
| Reporting | `Snapshot` `Sink` `NewRegistry` `Registry.Add` `Registry.Report` `NewSampledSink` `PerFingerprint` |
| Wire | `Encode` `Decode` `RegisterCodeMigration` · `json.Marshaler` |
| Diagnostics | `WithURL` `WithSource` `WithLabel` · `RegisterDoc` `DocFor` · `diag` subpackage |
| Barriers | `Opaque` |

Full generated reference: **[pkg.go.dev/github.com/ubgo/errx](https://pkg.go.dev/github.com/ubgo/errx)**.

## Integrations (contrib)

Each is a standalone module with its own README, step-by-step setup and runnable example:

| Module | Use it for | README |
|---|---|---|
| `contrib/httpx` | RFC 9457 `application/problem+json` HTTP responses (zero extra deps) | [docs](./contrib/httpx/README.md) |
| `contrib/grpc` | `google.rpc.Status` + `ErrorInfo`/`RetryInfo`/`LocalizedMessage`/`Help` | [docs](./contrib/grpc/README.md) |
| `contrib/connect` | ConnectRPC errors with typed details | [docs](./contrib/connect/README.md) |
| `contrib/graphql` | gqlgen `ErrorPresenter` (`extensions.code`) | [docs](./contrib/graphql/README.md) |
| `contrib/sentry` | Sentry capture with the **core fingerprint as the grouping key** | [docs](./contrib/sentry/README.md) |
| `contrib/otel` | OpenTelemetry exception span events + trace/span correlation | [docs](./contrib/otel/README.md) |
| `contrib/prometheus` | `errx_errors_total` metrics by stable identity | [docs](./contrib/prometheus/README.md) |
| `contrib/goerr` | Incremental migration from `github.com/ubgo/goerr` | [docs](./contrib/goerr/README.md) |

## Observability wiring

Wire sinks **explicitly, once, at startup** — no `init()` side effects:

```go
reg := errx.NewRegistry().
	Add(errx.NewSampledSink(sentryx.New(hub), errx.PerFingerprint(time.Minute, 5))).
	Add(otelx.NewSink()).
	Add(promx.New(prometheus.DefaultRegisterer))
otelx.Install() // trace/span ids auto-attach via Error.Context(ctx)

// in your error middleware:
reg.Report(ctx, err)
```

`SampledSink` rate-limits per fingerprint so one hot error cannot blow your Sentry bill, and surfaces an `errx.sampled_dropped` count instead of dropping silently.

## Diagnostics (miette-style)

```go
err := errx.New(`unknown column "emial"`).
	WithCode("PG_UNDEFINED_COLUMN").
	WithSource("query.sql", sql).
	WithLabel(strings.Index(sql, "emial"), 5, "no such column")
errx.RegisterDoc("PG_UNDEFINED_COLUMN", errx.DocEntry{
	URL:         "https://errors.example.com/PG_UNDEFINED_COLUMN",
	Remediation: `did you mean "email"?`,
})

fmt.Println(diag.String(err))
```

```
error[PG_UNDEFINED_COLUMN]: unknown column "emial"
  --> query.sql
3 | WHERE  emial = $1
  |        ^^^^^ no such column
help: did you mean "email"?
docs: https://errors.example.com/PG_UNDEFINED_COLUMN
```

`diag.Fprint(os.Stderr, err, diag.Auto())` auto-switches to a prose **narratable** form under `NO_COLOR` / `CI` / non-TTY (screen-reader friendly).

## Code generation — errxgen

Go has no derive macros. `cmd/errxgen` generates the boilerplate; generated code imports only `fmt`:

```go
//go:generate go run github.com/ubgo/errx/cmd/errxgen .

//errxgen: message="open %s: permission denied (uid %d)" args=Path,UID code=IO_DENIED unwrap=Err
type DeniedError struct {
	Path string
	UID  int
	Err  error
}
```

`go generate ./...` emits `Error()`, `Code()` and `Unwrap()`. Because `errx.Code`/`HasCode` honor any `Code() string` method, the generated type is first-class without being an `*errx.Error`.

## Optional `Result[T]` layer

For code that prefers railway style — **never required**, idiomatic `(value, error)` stays primary:

```go
import "github.com/ubgo/errx/result"

r := result.Try(func() (User, error) { return repo.Find(id) })
name := result.Map(r, func(u User) string { return u.Name }).UnwrapOr("anonymous")
```

## Testing helpers

```go
import "github.com/ubgo/errx/errtest"

errtest.Code(t, err, "NOT_FOUND")
errtest.Class(t, err, errx.ClassExpected)
errtest.Retryable(t, err, true)
errtest.InChain(t, err, io.ErrUnexpectedEOF)
errtest.Fingerprint(t, errA, errB) // assert two errors group together
```

## FAQ

**Is it really a drop-in over the standard library?** Yes. `errx.Wrap` works with `errors.Is`, `errors.As`, `errors.Join`. `*errx.Error` implements `Unwrap() []error`. You can adopt it incrementally.

**Does the core pull dependencies?** No. The core module is standard-library-only. A third-party dependency enters your build only when you import the matching `contrib` module.

**Will it bloat like `cockroachdb/errors`?** That is the explicit anti-goal. Core stays tiny; wire codec, redaction, and every vendor integration are opt-in. You pay for what you import.

**Is this a monad / `Result` framework?** No. `result` is an optional sub-package. The primary interface is idiomatic Go errors.

**Does it replace `github.com/ubgo/goerr`?** No — that thin module stays. `contrib/goerr` is a two-way migration bridge.

**Performance?** Stack capture is a single `runtime.Callers` storing `[]uintptr`, symbolized only when printed (~15 ns vs `pkg/errors` ~717 ns / 3 allocs). No capture happens unless you construct an errx error.

## Documentation

- [`docs/design.md`](./docs/design.md) — architecture: the `Report` seam, zero-dep core, no-`init()`, wrapping/class/stack rationale.
- [`docs/comparison.md`](./docs/comparison.md) — feature matrix vs the field, plus explicit non-goals and migration paths.
- [`docs/use-cases.md`](./docs/use-cases.md) — copy-pasteable patterns (HTTP, gRPC, validation, retry, cross-service, observability, diagnostics).
- [`docs/snippets.md`](./docs/snippets.md) — one-screen cheat sheet.
- Per-integration guides under [`contrib/*/README.md`](./contrib).

## Community

- [Contributing guide](./CONTRIBUTING.md) — repo layout, local setup, the checks every change must pass, how to add an adapter.
- [Code of Conduct](./CODE_OF_CONDUCT.md) — Contributor Covenant 2.1.
- [Security policy](./SECURITY.md) — private vulnerability reporting; redaction guarantees are in scope.
- [Changelog](./CHANGELOG.md) — what changed, per release.
- [Open an issue](https://github.com/ubgo/errx/issues) · [Discussions](https://github.com/ubgo/errx/discussions)

## License

Apache-2.0 — see [`LICENSE`](./LICENSE) and [`NOTICE`](./NOTICE). Part of the [`github.com/ubgo`](https://github.com/ubgo) ecosystem of small, focused, dependency-light Go libraries.
