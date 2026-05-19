# errx/contrib/connect — ConnectRPC errors for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/ubgo/errx/contrib/connect.svg)](https://pkg.go.dev/github.com/ubgo/errx/contrib/connect)

Convert any [`errx`](https://github.com/ubgo/errx) error into a **`*connect.Error`** ([ConnectRPC](https://connectrpc.com)) with a canonical Connect code and a typed **`google.rpc.ErrorInfo`** detail carrying the stable `(reason, domain)` identity and safe metadata. Works across Connect's JSON, gRPC, and gRPC-Web protocols.

## Contents

- [Why](#why)
- [Install](#install)
- [Step by step](#step-by-step)
- [Code mapping](#code-mapping)
- [API](#api)
- [Safety](#safety)

## Why

ConnectRPC speaks three wire protocols from one handler. `contrib/connect` ensures your `errx` identity, class, and safe fields serialize consistently across all of them, with the same `(reason, domain)` contract clients rely on — no per-handler error plumbing.

## Install

```sh
go get github.com/ubgo/errx/contrib/connect
```

Pulls `connectrpc.com/connect` and `google.golang.org/genproto/googleapis/rpc`.

## Step by step

1. **Return errx errors, converted, from your Connect handlers**:

   ```go
   import (
       "connectrpc.com/connect"
       "github.com/ubgo/errx"
       errxconnect "github.com/ubgo/errx/contrib/connect"
   )

   func (s *Server) GetThing(
       ctx context.Context, req *connect.Request[pb.GetThingRequest],
   ) (*connect.Response[pb.Thing], error) {
       t, err := s.store.Thing(ctx, req.Msg.Id)
       if err != nil {
           return nil, errxconnect.Error(ctx, errx.Wrap(err, "things: lookup").
               WithDomain("things").
               WithCode("NOT_FOUND").            // → connect.CodeNotFound
               WithPublic("Thing not found").     // → surfaced message
               WithSafe("thing_id", req.Msg.Id))  // → ErrorInfo.metadata
       }
       return connect.NewResponse(t), nil
   }
   ```

2. **Client side** — inspect the detail:

   ```go
   var cerr *connect.Error
   if errors.As(err, &cerr) {
       for _, d := range cerr.Details() {
           if v, derr := d.Value(); derr == nil {
               if info, ok := v.(*errdetails.ErrorInfo); ok {
                   log.Printf("reason=%s domain=%s", info.Reason, info.Domain)
               }
           }
       }
   }
   ```

## Code mapping

`errxconnect.CodeFor(err)` resolves the Connect code: explicit code mapping first, else by class (`ClassDefect → CodeInternal`, `ClassCancelled → CodeCanceled`, retryable expected `→ CodeUnavailable`, other expected `→ CodeInvalidArgument`).

| errx code | Connect code |
|---|---|
| `NOT_FOUND` | `CodeNotFound` |
| `ALREADY_EXISTS` | `CodeAlreadyExists` |
| `CONFLICT` | `CodeAborted` |
| `VALIDATION`, `INVALID_ARGUMENT` | `CodeInvalidArgument` |
| `UNAUTHENTICATED` | `CodeUnauthenticated` |
| `PERMISSION_DENIED`, `FORBIDDEN` | `CodePermissionDenied` |
| `RESOURCE_EXHAUSTED` | `CodeResourceExhausted` |
| `FAILED_PRECONDITION` | `CodeFailedPrecondition` |
| `UNAVAILABLE` | `CodeUnavailable` |
| `UNIMPLEMENTED` | `CodeUnimplemented` |
| `TIMEOUT` | `CodeDeadlineExceeded` |

## API

| Symbol | Purpose |
|---|---|
| `errxconnect.Error(ctx, err) *connect.Error` | Build the Connect error with `ErrorInfo` detail. The surfaced message is the end-user-safe message. |
| `errxconnect.CodeFor(err) connect.Code` | Resolve just the Connect code. |

## Safety

Only `WithSafe` fields are placed in `ErrorInfo.Metadata`; unsafe fields are dropped upstream by `errx.Snapshot`. The operator message never crosses the wire — only the public message.

See the [root README](../../README.md) and [`docs/use-cases.md`](../../docs/use-cases.md).
