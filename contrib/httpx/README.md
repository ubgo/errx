# errx/contrib/httpx — RFC 9457 problem+json error responses for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/ubgo/errx/contrib/httpx.svg)](https://pkg.go.dev/github.com/ubgo/errx/contrib/httpx) [![Go Report Card](https://goreportcard.com/badge/github.com/ubgo/errx/contrib/httpx)](https://goreportcard.com/report/github.com/ubgo/errx/contrib/httpx) [![CI](https://github.com/ubgo/errx/actions/workflows/ci-contrib-httpx.yml/badge.svg)](https://github.com/ubgo/errx/actions/workflows/ci-contrib-httpx.yml) [![license](https://img.shields.io/badge/license-Apache--2.0-blue)](../../LICENSE) ![Go](https://img.shields.io/badge/go-1.24%2B-00ADD8?logo=go) [![part of errx](https://img.shields.io/badge/part%20of-errx-6f42c1)](https://github.com/ubgo/errx)

Render any [`errx`](https://github.com/ubgo/errx) error as a standards-compliant **RFC 9457 / RFC 7807 `application/problem+json`** HTTP response — the JSON error body every modern Go API is converging on. Maps the error's stable `code` and class to the right HTTP status, exposes only the end-user-safe message, and **never leaks unsafe (unredacted) fields** to the client.

**Stdlib-only.** This module adds **zero third-party dependencies** — it consumes the neutral `errx.Report` and uses only `net/http` + `encoding/json`.

## Contents

- [Why](#why)
- [Install](#install)
- [Step by step](#step-by-step)
- [Full example](#full-example)
- [The Problem document](#the-problem-document)
- [Status mapping](#status-mapping)
- [Configuration](#configuration)
- [Safety / redaction](#safety--redaction)
- [Troubleshooting](#troubleshooting)

## Why

Hand-rolled `{"error": "..."}` bodies leak internals, have no stable machine code, and differ per service. RFC 9457 standardises `type` / `title` / `status` / `detail` / `instance` plus extension members. `httpx` produces it from your existing `errx` errors with one call — no per-handler mapping.

## Install

```sh
go get github.com/ubgo/errx/contrib/httpx
```

Requires Go 1.24+ and `github.com/ubgo/errx`.

## Step by step

1. **Return rich errors from your service layer** (do this once, everywhere):

   ```go
   func (s *Store) GetOrder(ctx context.Context, id string) (*Order, error) {
       o, err := s.db.Find(ctx, id)
       if errors.Is(err, sql.ErrNoRows) {
           return nil, errx.New("orders: no rows for "+id).
               WithDomain("orders").
               WithCode("NOT_FOUND").              // → HTTP 404 (see status table)
               WithPublic("Order not found").      // → problem.detail
               WithSafe("order_id", id)            // → problem.fields.order_id
       }
       if err != nil {
           return nil, errx.Wrap(err, "orders: query failed").WithCode("DB_READ")
       }
       return o, nil
   }
   ```

2. **Write the problem response in your handler / error middleware**:

   ```go
   func handler(w http.ResponseWriter, r *http.Request) {
       o, err := store.GetOrder(r.Context(), r.PathValue("id"))
       if err != nil {
           httpx.Write(w, r, err) // sets status + Content-Type + body
           return
       }
       json.NewEncoder(w).Encode(o)
   }
   ```

3. **(Optional) register custom code→status / code→type mappings at startup**:

   ```go
   httpx.RegisterStatus("RATE_LIMITED", http.StatusTooManyRequests)
   httpx.RegisterType("RATE_LIMITED", "https://errors.example.com/rate-limited")
   httpx.TypeBaseURL = "https://errors.example.com/" // default type URI prefix
   ```

## Full example

```go
package main

import (
	"errors"
	"net/http"

	"github.com/ubgo/errx"
	"github.com/ubgo/errx/contrib/httpx"
)

func main() {
	errx.RegisterDoc("NOT_FOUND", errx.DocEntry{URL: "https://errors.example.com/not-found"})

	http.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		err := errx.New("orders: no rows").
			WithDomain("orders").
			WithCode("NOT_FOUND").
			WithPublic("Order not found").
			WithLocalized("fr-FR", "Commande introuvable").
			With("internal_query", "SELECT …"). // unsafe → redacted, never sent
			WithSafe("order_id", "o-42")          // safe → included

		_ = errors.Is(err, err) // (your real lookup here)
		httpx.Write(w, r, err)
	})

	http.ListenAndServe(":8080", nil)
}
```

Response — `404 Not Found`, `Content-Type: application/problem+json`:

```json
{
  "type": "https://errors.example.com/not-found",
  "title": "NOT_FOUND",
  "status": 404,
  "detail": "Order not found",
  "instance": "/orders/o-42",
  "code": "NOT_FOUND",
  "fingerprint": "9f2a1c…",
  "docUrl": "https://errors.example.com/not-found",
  "localized": { "fr-FR": "Commande introuvable" },
  "fields": { "order_id": "o-42" }
}
```

`internal_query` is absent — unsafe fields are redacted upstream by `errx.Snapshot`.

## The Problem document

`httpx.FromError(ctx, err) (Problem, bool)` returns the document without writing it (useful for custom serialization or testing). `Problem` fields:

| Field | JSON | Source |
|---|---|---|
| `Type` | `type` | registered type URI, or `TypeBaseURL+code`, or `about:blank` |
| `Title` | `title` | the error `code`, or HTTP status text |
| `Status` | `status` | see [status mapping](#status-mapping) |
| `Detail` | `detail` | the **end-user-safe** message (`WithPublic`) |
| `Instance` | `instance` | request path |
| `Code` | `code` | `errx` stable code (extension member) |
| `Fingerprint` | `fingerprint` | deterministic grouping key |
| `TraceID` | `traceId` | propagated trace id, if any |
| `DocURL` | `docUrl` | from `WithURL` or the error-doc registry |
| `Localized` | `localized` | BCP-47 → message map |
| `Fields` | `fields` | **safe** fields only |

## Status mapping

`httpx.StatusFor(err)` resolves the HTTP status: an explicit code mapping first, otherwise derived from class — `ClassDefect → 500`, `ClassCancelled → 499`, retryable expected `→ 503`, other expected `→ 400`. Built-in conventional codes:

| errx code | HTTP |
|---|---|
| `NOT_FOUND` | 404 |
| `ALREADY_EXISTS`, `CONFLICT` | 409 |
| `VALIDATION` | 422 |
| `INVALID_ARGUMENT` | 400 |
| `UNAUTHENTICATED` | 401 |
| `PERMISSION_DENIED`, `FORBIDDEN` | 403 |
| `RESOURCE_EXHAUSTED` | 429 |
| `FAILED_PRECONDITION` | 412 |
| `UNAVAILABLE` | 503 |
| `UNIMPLEMENTED` | 501 |
| `TIMEOUT` | 504 |

Override or extend with `httpx.RegisterStatus(code, status)`.

## Configuration

| Symbol | Purpose |
|---|---|
| `httpx.Write(w, r, err) int` | Write the problem response; returns the status written. Falls back to 500 for non-errx errors. |
| `httpx.FromError(ctx, err) (Problem, bool)` | Build the document without writing. |
| `httpx.StatusFor(err) int` | Resolve the HTTP status only. |
| `httpx.RegisterStatus(code, status)` | Map an `errx` code to an HTTP status. |
| `httpx.RegisterType(code, uri)` | Map an `errx` code to a problem-type URI. |
| `httpx.TypeBaseURL` (var) | Default `type` URI prefix; `"about:blank"` disables it. |

All registration is explicit — call at startup, no `init()` magic.

## Safety / redaction

Only fields added with `WithSafe` cross to the client. Fields added with `With` are unsafe and are replaced with a redaction marker by `errx.Snapshot` *before* `httpx` ever sees them — leaking PII through the API is not possible by construction. The operator message (`.Error()`) is never sent; only `WithPublic` is.

## Troubleshooting

- **Body is `{"title":"Internal Server Error","status":500}`** — the error did not carry an `*errx.Error`. Wrap it: `errx.Wrap(err, "…").WithCode("…")`.
- **Wrong status** — register the mapping: `httpx.RegisterStatus("YOUR_CODE", http.StatusXxx)`.
- **`type` is `about:blank`** — set `httpx.TypeBaseURL` or call `httpx.RegisterType`.

See the [root README](../../README.md) and [`docs/use-cases.md`](../../docs/use-cases.md) for end-to-end patterns.
