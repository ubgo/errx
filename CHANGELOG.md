# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Pre-`v1.0.0`
the public API may change between minor versions.

## [Unreleased]

### Added — Phase 1 core (stdlib-only)

- `Error` type with stable identity (`domain` + `code`), `Class`
  (expected/defect/cancelled), `Severity`, separate operator and end-user
  message, `Hint`, `Owner`, retry metadata.
- Constructors: `New`, `Newf`, `Wrap`, `Wrapf`, `Note` (note-without-rewrap),
  `Join`.
- Transparent wrapping with inner-identity inheritance; explicit `Opaque()`
  barrier.
- Standard-library interop: `Unwrap() []error` (cause + suppressed), works
  with `errors.Is`/`errors.As`/`errors.Join`.
- Generic `As[T]`, plus `Get`, `Code`, `HasCode`, `FindByCode`, `ClassOf`,
  `IsExpected`/`IsDefect`/`IsCancelled`, `IsRetryable`, `RetryAfter`.
- Typed structured fields with per-field `Safe` flag; redaction enforced at
  `Snapshot`.
- Lazy single-`runtime.Callers` stack capture; `Frames()` symbolizes on
  demand.
- Suppressed (secondary) errors, kept matchable via `errors.Is`.
- Two-mode accumulation: `Accumulator`, `Accumulate`, `ParAccumulate`
  (goroutine-safe) with per-field paths.
- Deterministic, message-independent `Fingerprint` (FNV-1a of
  domain+code+origin frame).
- Neutral `Report` DTO + `Snapshot` + `Sink` interface + explicit `Registry`
  (no `init()` magic) — the single adapter seam.
- `fmt.Formatter` (`%s`/`%q`/`%+v`) and `slog.LogValuer`.
- `Recover`/`RecoverDo`/`Recovered`: panic → structured *Error
  (ClassDefect, SevFatal), original error preserved as cause.
- `RegisterContextExtractor` + `Error.Context(ctx)`: pluggable
  context→error enrichment (trace/span ids, request fields) with explicit
  registration, no `init()`.
- `errtest` subpackage: `Code`/`Class`/`Retryable`/`InChain`/`HasCode`/
  `NoError`/`Fingerprint` assertion helpers.
- Apache-2.0 license, `Taskfile.yml`, `.golangci.yml`, GitHub Actions CI
  (core + per-contrib-module), `go.work` for local multi-module dev.

### Added — Phase 2 contrib adapters (separate modules)

- `contrib/httpx` (stdlib-only): RFC 9457 `application/problem+json` —
  `Problem`, `FromError`, `StatusFor` (code + class → HTTP status),
  `Write`, `RegisterStatus`/`RegisterType`. Unsafe fields never leak
  (consumes the redacted `Report`).
- `contrib/otel`: OpenTelemetry `Sink` recording the error as an
  `exception` span event with exception.* semconv + span status Error;
  `Install()` registers a context extractor for trace/span ids.
- `contrib/sentry`: `Sink` capturing events with level from severity,
  code/domain tags, the **core fingerprint as the Sentry grouping key**,
  safe fields as context, `mechanism.handled=true`, oldest-first stack.
- `contrib/prometheus`: `Sink` incrementing `errx_errors_total`
  `{code,domain,severity,class}` — labels from stable identity (bounded
  cardinality; variable messages don't split the series).
- `contrib/grpc`: `Status`/`Err`/`CodeFor` — code+class → canonical gRPC
  code, with `ErrorInfo` (stable reason/domain + safe metadata),
  `RetryInfo`, `LocalizedMessage`, `Help` typed details.
- `codec` (core, stdlib-only): `Encode`/`Decode` + `json.Marshaler` —
  cross-service wire round-trip preserving identity/class/severity/retry/
  fingerprint so `Code`/`HasCode`/`IsRetryable`/`ClassOf` keep working
  after the hop; unsafe fields never serialized.
- `SampledSink` (core): per-fingerprint rate limiting (`PerFingerprint`)
  so one hot error can't blow the Sentry bill; surfaces a
  `errx.sampled_dropped` count on the next allowed report instead of
  silently dropping.
- `contrib/connect`: ConnectRPC `Error`/`CodeFor` — code+class → Connect
  code with `ErrorInfo` typed detail (safe metadata only).
- `contrib/graphql`: gqlgen `Present` ErrorPresenter — public message →
  Message, identity + safe fields → Extensions (`extensions.code`),
  internal detail never exposed.
- `contrib/goerr`: migration shim — `From` (goerr → errx, chain-aware)
  and `To` (errx → goerr) for incremental adoption; unsafe fields never
  cross the bridge.
- `OnError`/`OnSuccess` (Zig `errdefer` semantics) + `CloseSuppressing`
  / `AppendSuppressed`: fix the classic `defer f.Close()` masking bug —
  a failing Close is attached as a secondary error, never masking the
  primary.
- `result` subpackage (optional, generics): `Result[T]` with
  `Ok`/`Err`/`From`/`Try`, `Map`/`AndThen`/`MapErr`/`Recover`/`Match`/
  `Collect`, `Unwrap`/`UnwrapOr`. Never required — idiomatic
  `(value, error)` stays the primary interface.
- Diagnostic layer (miette-style): `WithURL`/`WithSource`/`WithLabel` on
  the core error, plus a stdlib-only `diag` subpackage with a graphical
  report handler (severity+code header, `-->` source pointer, source line
  with caret `^^^` underline + label, help, docs URL) and a narratable
  handler (prose, for CI / pipes / screen readers); `diag.Auto()` selects
  based on `NO_COLOR`/`CI`/TTY. Nothing in Go shipped this before.
- i18n: `WithLocalized(locale, msg)` / `Localized(locale, fallback)` with
  BCP-47 language fallback (fr-CA → fr-FR). Carried into `Report`, gRPC
  `LocalizedMessage` (one per locale), RFC 9457 `localized`, and GraphQL
  `extensions.localized`.
- Error documentation registry: `RegisterDoc(code, DocEntry{URL, Summary,
  Remediation})`; `URL()`/`Remediation()` resolve from the registry when
  not set explicitly (explicit always wins). Flows into `Report.DocURL`,
  gRPC `Help`, RFC 9457 `docUrl`, GraphQL `extensions.docUrl`, and the
  `diag` report.
- Return-trace: `Error.Trace()` and `Report.Trace` — one frame per wrap
  layer (innermost-first), the path the error traveled, distinct from the
  origin stack; shown in `%+v`. Cheap (one PC per `Wrap`), goroutine-safe.
- `codec`: `RegisterCodeMigration(old, new)` — `Decode` rewrites retired
  codes (chain + cycle guard) for cross-version-skew tolerance.
- Behavioral code detection: `Code`/`HasCode` now also honor any error
  exposing `Code() string`, so foreign / `errxgen`-generated typed errors
  are first-class without being `*errx.Error`.
- `cmd/errxgen`: AST-based generator (thiserror parity — Go has no derive
  macros). A `//errxgen: message=... args=... code=... unwrap=...`
  directive emits `Error()`/`Code()`/`Unwrap()`; generated code imports
  only `fmt`. Ships a runnable, CI-tested example; CI fails if generated
  code is stale.
- Runnable `Example_*` (pkg.go.dev docs + tests).
- Core test coverage ~94%; `errtest`/`result` 100%; every `contrib`
  module ≥90% (`httpx`/`connect`/`graphql`/`prometheus` 100%, `goerr`
  98%, `otel` 94%, `grpc` 92%, `sentry` 91%).
- `contrib/otel` now requires **Go 1.25+** (tracks OpenTelemetry v1.43+);
  its CI matrix uses Go 1.25. All other modules remain Go 1.24.
- Per-contrib-module GitHub Actions CI (8 modules); `go.work` is
  local-dev only (git-ignored) so each module builds standalone
  (ecosystem rule #5).
