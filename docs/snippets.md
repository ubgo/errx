# errx snippets — one-screen cheat sheet

Copy-paste recipes. Full prose in the [root README](../README.md) and [`use-cases.md`](./use-cases.md).

## Create / wrap

```go
errx.New("db: deadlock").WithCode("TX_RETRY").WithDomain("billing")
errx.Newf("user %s not found", id).WithCode("NOT_FOUND")
errx.Wrap(err, "load config").WithCode("CONFIG")           // transparent (Is/As traverse)
errx.Wrapf(err, "load %s", path)
errx.Wrap(err, "boundary").Opaque()                        // barrier: hide cause from Is/As
errx.Note(err, "attempt", 3)                               // breadcrumb, no new frame
errx.Join(err1, err2, err3)                                // multi-error
```

## Identity / class / severity / retry

```go
errx.New("x").
    WithDomain("billing").WithCode("CARD_DECLINED").
    WithClass(errx.ClassExpected).            // ClassDefect | ClassCancelled
    WithSeverity(errx.SevWarn).               // SevDebug|Info|Warn|Error|Fatal
    WithRetryable(3 * time.Second).
    WithOwner("payments")
```

## Inspect (works anywhere in the chain)

```go
errx.Code(err)                 // "" if none; also reads any Code() string method
errx.HasCode(err, "NOT_FOUND")
errx.FindByCode(err, "X")      // *errx.Error or nil
e, ok := errx.As[*MyErr](err)  // generic errors.As
errx.Get(err)                  // first *errx.Error, or nil
errx.ClassOf(err)              // context.Canceled → ClassCancelled
errx.IsExpected(err) / IsDefect(err) / IsCancelled(err)
errx.IsRetryable(err) / errx.RetryAfter(err)
errx.Fingerprint(err)          // stable grouping key
```

## Messages

```go
e := errx.Get(err)
e.Public("Something went wrong")     // end-user-safe, never operator detail
e.WithLocalized("fr-FR", "…")
e.Localized("fr-CA", "fb")           // language fallback fr-CA → fr-FR
e.Remediation()                      // hint, or registry remediation
```

## Fields & redaction

```go
errx.New("x").
    WithSafe("order_id", id).  // survives to logs / problem+json / Sentry / wire
    With("card_pan", pan)      // UNSAFE → ‹redacted› at every boundary
```

## Suppressed errors & error-path cleanup

```go
defer errx.CloseSuppressing(&err, f)            // close err = secondary, never masks
defer errx.OnError(&err, func(){ rollback() })  // failure path only (Zig errdefer)
defer errx.OnSuccess(&err, commit)
errx.AppendSuppressed(&err, extra)
errx.Get(err).Suppress(closeErr)
```

## Accumulate (collect every failure)

```go
acc := errx.NewAccumulator()
acc.Add("email", validateEmail(in.Email))
acc.AddErr(checkQuota(ctx))
return acc.ErrorOrNil()                 // one VALIDATION error w/ field paths
errx.Accumulate(fnA, fnB)               // fail-soft, sequential
errx.ParAccumulate(fnA, fnB)            // concurrent
```

## Panic → error

```go
func h() (err error) { defer errx.Recover(&err); return risky() }
err := errx.RecoverDo(func() error { return risky() })
errx.Recovered(err)                     // captured from a panic?
```

## Cross-service wire

```go
blob, _ := errx.Encode(err)             // unsafe fields NOT serialized
got,  _ := errx.Decode(blob)            // typed *errx.Error rebuilt
errx.RegisterCodeMigration("OLD", "NEW")
```

## Observability (wire once at startup)

```go
reg := errx.NewRegistry().
    Add(errx.NewSampledSink(sentryx.New(nil), errx.PerFingerprint(time.Minute, 5))).
    Add(otelx.NewSink()).
    Add(promx.New(prometheus.DefaultRegisterer))
otelx.Install()
reg.Report(ctx, err)                    // fan-out, redacted, fingerprinted
```

## Transport one-liners

```go
httpx.Write(w, r, err)                       // RFC 9457 problem+json
return errxgrpc.Err(ctx, err)                // google.rpc.Status + details
return nil, errxconnect.Error(ctx, err)      // *connect.Error
srv.SetErrorPresenter(graphqlx.Present)      // gqlgen
```

## Diagnostics

```go
errx.New(`unknown column "emial"`).
    WithCode("PG_UNDEFINED_COLUMN").
    WithSource("query.sql", sql).
    WithLabel(strings.Index(sql, "emial"), 5, "no such column")
fmt.Println(diag.String(err))                // carets; or diag.Fprint(os.Stderr, err, diag.Auto())
errx.RegisterDoc("PG_UNDEFINED_COLUMN", errx.DocEntry{URL: "…", Remediation: "…"})
```

## Test helpers

```go
errtest.Code(t, err, "NOT_FOUND")
errtest.Class(t, err, errx.ClassExpected)
errtest.Retryable(t, err, true)
errtest.InChain(t, err, io.EOF)
errtest.Fingerprint(t, a, b)                 // assert same grouping
```

## errxgen (codegen)

```go
//go:generate go run github.com/ubgo/errx/cmd/errxgen .

//errxgen: message="open %s: denied (uid %d)" args=Path,UID code=IO_DENIED unwrap=Err
type DeniedError struct { Path string; UID int; Err error }
```
