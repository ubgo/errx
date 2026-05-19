# Design

## Goal

The research behind `errx` surveyed ~90 error libraries across 20+ languages. The finding: no library — in any language — ships the coherent union of stable identity + expected/defect/cancelled class + public/internal split + typed redactable fields + retry/severity + lazy stack + suppressed errors + accumulation + deterministic fingerprint + miette-grade diagnostics + i18n + wire interop, behind a zero-dependency core. `errx` is that union.

## The one rule: the `Report` seam

Every observability/transport integration consumes exactly one neutral, serializable type — `errx.Report` — produced by `Snapshot`. Adapters never touch `*errx.Error` internals; the core never imports an adapter.

```
            ┌─────────────┐   Snapshot(ctx,err)   ┌──────────┐
  *Error ──▶│  errx core  │──────────────────────▶│  Report  │
            │ (stdlib only)│  (redaction applied) └────┬─────┘
            └─────────────┘                            │ read-only
                                                       ▼
           contrib/{http,grpc,connect,graphql,sentry,otel,prometheus,goerr}
```

Consequences:

- **You pay for what you import.** The core has zero third-party dependencies. Each `contrib/*` is a separate module with its own `go.mod`, version, and CI.
- **Redaction happens once**, in `Snapshot`, before any adapter sees the data. A field not marked `Safe` is replaced with a redaction marker — safe-by-construction, without cockroach's pervasive redactable-string type.
- **One fingerprint everywhere.** Computed once in the core (FNV-1a of domain+code+origin frame, message excluded) and reused by Sentry grouping, the Prometheus label, the log field, and the wire codec — solving the "four disconnected grouping mechanisms" problem.

## No `init()` magic

Every registry (`Sink` registry, context extractors, the doc registry, HTTP status map) is populated by an explicit call the application makes at startup. Importing a package never changes behavior. This is an ecosystem rule and it makes test isolation and audit trivial.

## Wrapping semantics

`Wrap` is **transparent** by default: `errors.Is/As` traverse the cause, and an inner `*errx.Error`'s identity/class is inherited so an outer annotation never erases precise inner classification. `.Opaque()` is the deliberate exception — a barrier that hides a dependency's sentinels from your package's API surface. (Chain inspection is table-stakes for a structured error type; opacity is the special case, not the default.)

## Class vs Severity

- **Class** (`Expected` / `Defect` / `Cancelled`) drives *control flow*: 4xx + maybe retry vs alert + 5xx vs stay-quiet. This is the single highest-leverage field; idiomatic Go blurs all three behind `error`.
- **Severity** (`Debug…Fatal`) drives *log/alert routing*. `Fatal` is a severity, not a class — keeping the class enum at exactly three orthogonal values.

## Stack capture cost

A single `runtime.Callers` at construction stores `[]uintptr`; symbolization is deferred to `Frames()` (only when something prints). This is the answer to the universal criticism of `pkg/errors` (≈717ns / 3 allocs eagerly on every wrap). No capture happens unless you construct an `errx` error.

## Suppressed errors

`Unwrap() []error` returns `[cause, suppressed...]`, so `errors.Is/As` match both, but the primary and secondary are kept distinct (`Cause()` vs `Suppressed()`). `CloseSuppressing` folds a failing `Close()` into the secondary slot — the classic `defer f.Close()` masking bug, which Go's stdlib has no remedy for.

## Modules

| Module | Deps | Role |
|---|---|---|
| `github.com/ubgo/errx` | none | core, `codec`, `SampledSink`, `diag`, `result`, `errtest` |
| `contrib/httpx` | none | RFC 9457 problem+json |
| `contrib/grpc` | grpc, genproto | `google.rpc.Status` + typed details |
| `contrib/connect` | connect-go | ConnectRPC error |
| `contrib/graphql` | gqlparser | gqlgen presenter |
| `contrib/sentry` | sentry-go | capture, core fingerprint = grouping key |
| `contrib/otel` | otel | exception span event + ctx extractor |
| `contrib/prometheus` | client_golang | `errx_errors_total` |
| `contrib/goerr` | ubgo/goerr | migration bridge |
