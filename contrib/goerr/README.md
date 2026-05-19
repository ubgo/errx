# errx/contrib/goerr — migrate from github.com/ubgo/goerr to errx

[![Go Reference](https://pkg.go.dev/badge/github.com/ubgo/errx/contrib/goerr.svg)](https://pkg.go.dev/github.com/ubgo/errx/contrib/goerr) [![Go Report Card](https://goreportcard.com/badge/github.com/ubgo/errx/contrib/goerr)](https://goreportcard.com/report/github.com/ubgo/errx/contrib/goerr) [![CI](https://github.com/ubgo/errx/actions/workflows/ci-contrib-goerr.yml/badge.svg)](https://github.com/ubgo/errx/actions/workflows/ci-contrib-goerr.yml) [![license](https://img.shields.io/badge/license-Apache--2.0-blue)](../../LICENSE) ![Go](https://img.shields.io/badge/go-1.24%2B-00ADD8?logo=go) [![part of errx](https://img.shields.io/badge/part%20of-errx-6f42c1)](https://github.com/ubgo/errx)

A **two-way migration bridge** between the thin, shipped [`github.com/ubgo/goerr`](https://github.com/ubgo/goerr) and the full [`github.com/ubgo/errx`](https://github.com/ubgo/errx). Adopt `errx` incrementally — package by package — without a big-bang rewrite and without breaking code still written against `goerr`.

## Contents

- [Why](#why)
- [Install](#install)
- [Migration strategy](#migration-strategy)
- [`From` — goerr → errx](#from--goerr--errx)
- [`To` — errx → goerr](#to--errx--goerr)
- [Field safety across the bridge](#field-safety-across-the-bridge)
- [API](#api)

## Why

`goerr` is small and already in production in some services. Rewriting everything at once is risky. This bridge lets a new module use the full `errx` model while still interoperating with callers and dependencies that produce or expect `goerr` — so the migration can be gradual and reversible.

## Install

```sh
go get github.com/ubgo/errx/contrib/goerr
```

Pulls `github.com/ubgo/goerr` and `github.com/ubgo/errx`.

## Migration strategy

1. **New / refactored code** uses `errx` directly.
2. **At the boundary where inbound `goerr` arrives** (a dependency you haven't migrated), call `shim.From(err)` to lift it into a full `*errx.Error` — identity, messages, trace id, severity and key/values are preserved.
3. **At the boundary where a not-yet-migrated caller expects `goerr`**, call `shim.To(err)` to hand back a `*goerr.Error`.
4. Delete the bridge calls module-by-module as both sides move to `errx`.

```go
import shim "github.com/ubgo/errx/contrib/goerr"

// inbound: dependency still returns *goerr.Error
func (s *Service) Handle(ctx context.Context) error {
    if err := legacy.Do(ctx); err != nil {
        return errx.Wrap(shim.From(err), "service: handle"). // now full errx
            WithOwner("payments")
    }
    return nil
}

// outbound: a caller still expects *goerr.Error
func Adapt(err error) *goerr.Error { return shim.To(err) }
```

## `From` — goerr → errx

`shim.From(err) *errx.Error` finds a `*goerr.Error` **anywhere in the chain** (`errors.As`) and maps:

| goerr | errx |
|---|---|
| `Message` | operator message |
| `MessageUser` | `WithPublic` |
| `Code` | `WithCode` |
| `TraceID` | `WithTrace(id, "")` |
| `Level` | `WithSeverity` (`debug/info/warn/error/fatal`) |
| `KV` map | unsafe fields (`With`) |
| `Data` | `With("data", …)` |

If no `goerr` is present the error is wrapped as-is. `From(nil)` returns `nil`.

## `To` — errx → goerr

`shim.To(err) *goerr.Error` maps an `errx` error back:

| errx | goerr |
|---|---|
| operator message | `Message` |
| `Public()` (or operator if unset) | `MessageUser` |
| `Code()` | `WithCode` |
| `SeverityOf()` | `WithLevel` |
| `TraceID()` | `WithTraceID` |
| **safe** fields | `WithKVMap` |

`To(nil)` returns `nil`.

## Field safety across the bridge

`To` copies **only `WithSafe` fields** into goerr `KV`. Unsafe fields are dropped, never silently exposed through the bridge. `From` brings goerr `KV`/`Data` in as **unsafe** fields (redacted at trust boundaries by default) — mark them `WithSafe` explicitly if they are non-sensitive and you want them surfaced.

## API

| Symbol | Purpose |
|---|---|
| `shim.From(err error) *errx.Error` | Lift inbound goerr (chain-aware) into a full errx error. |
| `shim.To(err error) *goerr.Error` | Convert an errx error back to goerr for un-migrated callers. |

See the [root README](../../README.md) for the full `errx` model.
