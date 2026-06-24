# AGENTS.md — codebase map for AI agents

Read this first. Orientation map for `ubgo/errx` so a fresh agent knows what every part does and where to change things, without reading every file.

## What this repo is

`ubgo/errx` is **one coherent structured error type for Go**: a `*Error` carrying stable machine identity (domain + code), expected/defect/cancelled classification, a user-vs-operator message split, typed structured fields (redactable by default), lazy single-alloc stack capture, return-trace, retry/severity metadata, and miette-style diagnostics. It's drop-in over stdlib `Is`/`As`/`Join`/`Unwrap`. Every observability/transport integration (Sentry, OTEL, gRPC, ConnectRPC, GraphQL, HTTP problem+json, Prometheus, slog) is a separate explicitly-registered `contrib/` adapter reading one neutral `Report` DTO — no `init()` magic. The core is **zero-dependency**. See `README.md` and `docs/`.

## Modules

| Path | Module | Role |
|---|---|---|
| `.` | `github.com/ubgo/errx` | The `*Error`, construction, codec, report seam, slog, diagnostics. Zero deps. |
| `contrib/{sentry,otel,grpc,connect,graphql,httpx,prometheus,goerr}` | each own module | Adapters: each consumes a `Report` and emits to its target (or, for `goerr`, a migration shim from the older lib). |
| `cmd/errxgen` | tool | Code generator for error-code catalogs/docs. |

Go 1.24. Adapters are isolated so the core stays dependency-free.

## Core files — what each owns

| File | Responsibility |
|---|---|
| `errx.go` | The `*Error` type, its accessors (`Domain`/`Code`/`Class`/`Severity`/`Public`/…) and the `With*` builder methods. |
| `construct.go` | Constructors: `New`, `Newf`, `Wrap`, `Wrapf`, `Note`, `Join`. |
| `ctx.go` | `ContextExtractor` + `RegisterContextExtractor` — pull trace/span/fields from `context.Context`. |
| `report.go` | The single seam: `Report` (neutral, redacted DTO), `Snapshot`, `Sink`, `Registry` (explicit fan-out to adapters). |
| `codec.go` | `Encode`/`Decode` — wire round-trip; `RegisterCodeMigration`. |
| `slog.go` | `LogValue` — structured logging integration. |
| `stack.go` | Lazy single-alloc stack capture + `Frame`/`Trace`. |
| `format.go` | `fmt.Formatter` (`%v`/`%+v` verbose output). |
| `fingerprint.go` | Stable fingerprint for grouping/dedup. |
| `inspect.go` | Walk/inspect the wrapped error chain. |
| `accumulate.go` | `Accumulator` + `Accumulate`/`ParAccumulate` — collect many errors. |
| `errdefer.go` | `OnError` — run cleanup only on failure. |
| `recover.go` | Panic-to-`*Error` recovery helpers. |
| `sample.go` | `SampledSink` — rate-limit reporting per fingerprint. |
| `docreg.go` | `RegisterDoc` / `DocEntry` — attach remediation docs to codes. |

| Dir | Role |
|---|---|
| `diag/` | miette-style rich diagnostics (source snippets, labels). |
| `result/` | A `Result[T]`-style helper. |
| `errtest/` | Test helpers/assertions for errx errors. |

## The seam to understand

Everything observability/transport flows through **one DTO**: `report.go:Snapshot(ctx, err)` projects the nearest `*Error` into a redacted `Report`; a `Registry` fans that `Report` out to registered `Sink`s. Adapters in `contrib/*` only ever consume a `Report` — they never touch `*Error` internals, and the core never imports an adapter. Redaction happens once, in `Snapshot`, so any sink crossing a trust boundary is safe by construction.

## Conventions

- **Zero-dependency core; adapters are isolated modules**, registered explicitly (no `init()` magic).
- **Redaction by default** — fields are redacted in the `Report` unless marked safe.
- Drop-in over stdlib `errors` (`Is`/`As`/`Join`/`Unwrap([]error)`).
- Comments explain *why*, not *what*.

## Where to look for X

- "Create/wrap an error" → `construct.go`.
- "Send errors to Sentry/OTEL/gRPC/…" → `report.go` (the `Report`/`Registry`/`Sink` seam) + `contrib/<target>`.
- "Trace correlation" → `ctx.go` (`RegisterContextExtractor`).
- "Wire format / encode-decode" → `codec.go`.
- "Generate an error-code catalog" → `cmd/errxgen`.
