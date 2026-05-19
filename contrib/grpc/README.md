# errx/contrib/grpc — google.rpc.Status + error details for Go gRPC

[![Go Reference](https://pkg.go.dev/badge/github.com/ubgo/errx/contrib/grpc.svg)](https://pkg.go.dev/github.com/ubgo/errx/contrib/grpc) [![Go Report Card](https://goreportcard.com/badge/github.com/ubgo/errx/contrib/grpc)](https://goreportcard.com/report/github.com/ubgo/errx/contrib/grpc) [![CI](https://github.com/ubgo/errx/actions/workflows/ci-contrib-grpc.yml/badge.svg)](https://github.com/ubgo/errx/actions/workflows/ci-contrib-grpc.yml) [![license](https://img.shields.io/badge/license-Apache--2.0-blue)](../../LICENSE) ![Go](https://img.shields.io/badge/go-1.24%2B-00ADD8?logo=go) [![part of errx](https://img.shields.io/badge/part%20of-errx-6f42c1)](https://github.com/ubgo/errx)

Map any [`errx`](https://github.com/ubgo/errx) error to a canonical **gRPC `status.Status`** with **typed `google.rpc` error details** — `ErrorInfo` (the stable `(reason, domain)` identity), `RetryInfo`, `LocalizedMessage`, and `Help`. The end-user-safe message becomes the status message; unredacted fields never cross the wire.

## Contents

- [Why](#why)
- [Install](#install)
- [Step by step](#step-by-step)
- [Full example](#full-example)
- [Code mapping](#code-mapping)
- [Error details emitted](#error-details-emitted)
- [API](#api)
- [Safety](#safety)

## Why

gRPC's rich error model (`google.rpc.Status` + `errdetails`) is the industry standard for machine-readable RPC errors, but wiring it by hand per handler is tedious and inconsistent. `contrib/grpc` derives the whole thing from your `errx` identity in one call, so clients get a stable `(reason, domain)`, retry timing, and a localized message automatically.

## Install

```sh
go get github.com/ubgo/errx/contrib/grpc
```

Pulls `google.golang.org/grpc` and `google.golang.org/genproto/googleapis/rpc`.

## Step by step

1. **Return errx errors from your service methods**:

   ```go
   func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
       u, err := s.store.User(ctx, req.Id)
       if err != nil {
           return nil, errxgrpc.Err(ctx, errx.Wrap(err, "users: lookup").
               WithDomain("users").
               WithCode("NOT_FOUND").                 // → codes.NotFound
               WithPublic("User not found").           // → status message + LocalizedMessage
               WithSafe("user_id", req.Id))            // → ErrorInfo.metadata
       }
       return u, nil
   }
   ```

   `errxgrpc.Err(ctx, err)` returns a ready `error` (`nil` if `err` is `nil`) — return it directly.

2. **Client side** — read the typed details back:

   ```go
   st := status.Convert(err)
   for _, d := range st.Details() {
       switch info := d.(type) {
       case *errdetails.ErrorInfo:
           log.Printf("reason=%s domain=%s", info.Reason, info.Domain)
       case *errdetails.RetryInfo:
           time.Sleep(info.RetryDelay.AsDuration())
       }
   }
   ```

## Full example

```go
e := errx.New("pq: deadlock on user_orders").
	WithDomain("billing").
	WithCode("TX_FAIL").
	WithPublic("Please retry").
	WithHint("use exponential backoff").
	WithRetryable(2 * time.Second).
	WithLocalized("fr-FR", "Veuillez réessayer").
	With("internal_sql", "…").  // unsafe → dropped
	WithSafe("order_id", "o-9")  // safe → ErrorInfo.metadata

st := errxgrpc.Status(context.Background(), e)
// st.Code()    == codes.Unavailable   (retryable expected error)
// st.Message() == "Please retry"
// st.Details() contains ErrorInfo{Reason:"TX_FAIL", Domain:"billing",
//   Metadata:{"order_id":"o-9"}}, RetryInfo{2s},
//   LocalizedMessage{"fr-FR","Veuillez réessayer"}, Help{…hint…}
return st.Err()
```

## Code mapping

`errxgrpc.CodeFor(err)` resolves the canonical `codes.Code`: explicit code mapping first, else by class (`ClassDefect → Internal`, `ClassCancelled → Canceled`, retryable expected `→ Unavailable`, other expected `→ InvalidArgument`).

| errx code | gRPC code |
|---|---|
| `NOT_FOUND` | `NotFound` |
| `ALREADY_EXISTS` | `AlreadyExists` |
| `CONFLICT` | `Aborted` |
| `VALIDATION`, `INVALID_ARGUMENT` | `InvalidArgument` |
| `UNAUTHENTICATED` | `Unauthenticated` |
| `PERMISSION_DENIED`, `FORBIDDEN` | `PermissionDenied` |
| `RESOURCE_EXHAUSTED` | `ResourceExhausted` |
| `FAILED_PRECONDITION` | `FailedPrecondition` |
| `UNAVAILABLE` | `Unavailable` |
| `UNIMPLEMENTED` | `Unimplemented` |
| `TIMEOUT` | `DeadlineExceeded` |

## Error details emitted

| Detail | From |
|---|---|
| `ErrorInfo{Reason, Domain, Metadata}` | `code`, `domain`, and **safe** fields |
| `RetryInfo{RetryDelay}` | `WithRetryable(d)` |
| `LocalizedMessage{Locale, Message}` | each `WithLocalized` entry, else the public message as `en-US` |
| `Help{Links:[{Description, Url}]}` | `WithHint` / `Remediation` + `WithURL` / error-doc registry |

## API

| Symbol | Purpose |
|---|---|
| `errxgrpc.Status(ctx, err) *status.Status` | Build the full status + details. |
| `errxgrpc.Err(ctx, err) error` | `Status(...).Err()`; `nil` if `err` is `nil`. Return from handlers. |
| `errxgrpc.CodeFor(err) codes.Code` | Resolve just the gRPC code. |

## Safety

Only `WithSafe` fields appear in `ErrorInfo.Metadata`. Unsafe fields are dropped by `errx.Snapshot` before this module runs — sensitive data cannot reach the wire. The operator message is never sent; the public/localized message is.

See the [root README](../../README.md) for the full error model and [`docs/use-cases.md`](../../docs/use-cases.md) for end-to-end gRPC patterns.
