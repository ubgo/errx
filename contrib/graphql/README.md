# errx/contrib/graphql — gqlgen error presenter for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/ubgo/errx/contrib/graphql.svg)](https://pkg.go.dev/github.com/ubgo/errx/contrib/graphql) [![Go Report Card](https://goreportcard.com/badge/github.com/ubgo/errx/contrib/graphql)](https://goreportcard.com/report/github.com/ubgo/errx/contrib/graphql) [![CI](https://github.com/ubgo/errx/actions/workflows/ci-contrib-graphql.yml/badge.svg)](https://github.com/ubgo/errx/actions/workflows/ci-contrib-graphql.yml) [![license](https://img.shields.io/badge/license-Apache--2.0-blue)](../../LICENSE) ![Go](https://img.shields.io/badge/go-1.24%2B-00ADD8?logo=go) [![part of errx](https://img.shields.io/badge/part%20of-errx-6f42c1)](https://github.com/ubgo/errx)

Render any [`errx`](https://github.com/ubgo/errx) error as a **GraphQL error** for [gqlgen](https://gqlgen.com): the end-user-safe message becomes `message`, the stable identity and safe fields go under `extensions` (conventional `extensions.code`), and internal detail is never exposed.

## Contents

- [Why](#why)
- [Install](#install)
- [Step by step](#step-by-step)
- [Extensions emitted](#extensions-emitted)
- [API](#api)
- [Safety](#safety)

## Why

GraphQL clients branch on `extensions.code`, not on a human string. Without a consistent presenter, every resolver invents its own error shape and risks leaking internals into `message`. `contrib/graphql` gives every resolver the same machine-readable, redaction-safe shape from your `errx` errors.

## Install

```sh
go get github.com/ubgo/errx/contrib/graphql
```

Pulls `github.com/vektah/gqlparser/v2` (the error type gqlgen uses).

## Step by step

1. **Install the presenter on your gqlgen server** (once, at startup):

   ```go
   import (
       "github.com/99designs/gqlgen/graphql/handler"
       graphqlx "github.com/ubgo/errx/contrib/graphql"
   )

   srv := handler.NewDefaultServer(es)
   srv.SetErrorPresenter(func(ctx context.Context, e error) *gqlerror.Error {
       return graphqlx.Present(ctx, e)
   })
   ```

2. **Return errx errors from resolvers** — nothing special needed:

   ```go
   func (r *queryResolver) Order(ctx context.Context, id string) (*model.Order, error) {
       o, err := r.store.Order(ctx, id)
       if err != nil {
           return nil, errx.New("orders: not found").
               WithDomain("orders").
               WithCode("NOT_FOUND").
               WithPublic("Order not found").
               With("internal", "…").       // unsafe → never in extensions
               WithSafe("order_id", id)      // safe → extensions.fields.order_id
       }
       return o, nil
   }
   ```

   Response:

   ```json
   {
     "errors": [{
       "message": "Order not found",
       "extensions": {
         "code": "NOT_FOUND",
         "domain": "orders",
         "class": "expected",
         "fingerprint": "9f2a1c…",
         "fields": { "order_id": "o-42" }
       }
     }]
   }
   ```

## Extensions emitted

| Key | Source |
|---|---|
| `code` | `errx` stable code |
| `domain` | `errx` domain |
| `class` | `expected` / `defect` / `cancelled` |
| `fingerprint` | deterministic grouping key |
| `retryable`, `retryAfter` | `WithRetryable` |
| `traceId` | propagated trace id |
| `docUrl` | `WithURL` or error-doc registry |
| `localized` | BCP-47 → message map |
| `fields` | **safe** fields only |

`message` is the end-user-safe message (`WithPublic`), or `"Internal server error"` if none was set. Non-errx errors fall back to a well-formed generic GraphQL error.

## API

| Symbol | Purpose |
|---|---|
| `graphqlx.Present(ctx, err) *gqlerror.Error` | Build the GraphQL error. Use as a gqlgen `ErrorPresenter`. |

## Safety

Only `WithSafe` fields enter `extensions.fields`; unsafe fields are redacted upstream by `errx.Snapshot`. The operator message is never placed in `message`.

See the [root README](../../README.md) and [`docs/use-cases.md`](../../docs/use-cases.md).
