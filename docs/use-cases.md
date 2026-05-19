# Use cases

Worked, copy-pasteable patterns. Every example keeps internal detail out of the client and unsafe fields out of logs/wire automatically.

## 1. Service-layer error → HTTP response

```go
// service layer
func (s *Store) GetOrder(ctx context.Context, id string) (*Order, error) {
	o, err := s.db.Find(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errx.New("order not found in db").
			WithDomain("orders").WithCode("NOT_FOUND").
			WithPublic("Order not found").
			WithSafe("order_id", id)
	}
	if err != nil {
		return nil, errx.Wrap(err, "orders: query failed").WithCode("DB_READ")
	}
	return o, nil
}

// transport layer
func handler(w http.ResponseWriter, r *http.Request) {
	o, err := store.GetOrder(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Write(w, r, err) // → 404 application/problem+json, "Order not found"
		return
	}
	json.NewEncoder(w).Encode(o)
}
```

The DB error message never reaches the client; `NOT_FOUND` maps to 404; `order_id` is a safe field so it appears in the problem document, anything unsafe would be redacted.

## 2. Form validation — collect every error

```go
func validate(in SignupInput) error {
	acc := errx.NewAccumulator()
	acc.Add("name", nonEmpty(in.Name))
	acc.Add("email", validEmail(in.Email))
	acc.Add("age", inRange(in.Age, 13, 120))
	return acc.ErrorOrNil() // nil, or one VALIDATION error listing all bad fields
}
```

Returns a single error with `Code() == "VALIDATION"`; `httpx` maps it to 422, each field path is a safe field.

## 3. Background worker — don't let Close() mask the failure

```go
func process(path string) (err error) {
	f, err := os.Open(path)
	if err != nil {
		return errx.Wrap(err, "open input").WithCode("IO")
	}
	defer errx.CloseSuppressing(&err, f) // close error becomes secondary, never masks
	defer errx.OnError(&err, func() { metrics.Failed.Inc() }) // only on failure path
	return parse(f)
}
```

If `parse` fails *and* `Close` fails, the parse error stays primary; the close error is attached as suppressed and still `errors.Is`-matchable.

## 4. Retry that consults the error

```go
for attempt := 0; ; attempt++ {
	err := call(ctx)
	if err == nil || !errx.IsRetryable(err) || attempt == maxAttempts {
		return err
	}
	wait := errx.RetryAfter(err)
	if wait == 0 {
		wait = backoff(attempt)
	}
	time.Sleep(wait)
}
```

The error itself answers "retry?" and "after how long?" — no type switches.

## 5. Cross-service propagation

```go
// producer service
return errx.Encode(err) // []byte JSON; unsafe fields are NOT serialized

// consumer service
e, _ := errx.Decode(body)
if errx.HasCode(e, "QUOTA_EXCEEDED") { ... }      // identity survived the hop
if errx.IsRetryable(e) { retryLater(errx.RetryAfter(e)) }
// e.Fingerprint() == the producer's fingerprint → groups together in Sentry
```

## 6. Observability wiring (once, at startup)

```go
reg := errx.NewRegistry().
	Add(errx.NewSampledSink(sentryx.New(hub), errx.PerFingerprint(time.Minute, 5))).
	Add(otelx.NewSink()).
	Add(promx.New(prometheus.DefaultRegisterer))
otelx.Install() // trace/span ids auto-attach via Error.Context(ctx)

// in your error middleware
reg.Report(ctx, err)
```

One hot error can no longer blow the Sentry bill (sampled per fingerprint); the same fingerprint groups it in Sentry, Prometheus and logs.

## 7. Compiler-grade diagnostic for a DSL/config error

```go
e := errx.New(`unknown column "emial"`).
	WithCode("PG_UNDEFINED_COLUMN").
	WithSource("query.sql", sql).
	WithLabel(strings.Index(sql, "emial"), 5, "no such column")
errx.RegisterDoc("PG_UNDEFINED_COLUMN", errx.DocEntry{
	URL:         "https://errors.example.com/PG_UNDEFINED_COLUMN",
	Remediation: `did you mean "email"?`,
})

diag.Fprint(os.Stderr, e, diag.Auto())
```

```
error[PG_UNDEFINED_COLUMN]: unknown column "emial"
  --> query.sql
3 | WHERE  emial = $1
  |        ^^^^^ no such column
help: did you mean "email"?
docs: https://errors.example.com/PG_UNDEFINED_COLUMN
```

Under `CI`/`NO_COLOR`/pipe it auto-switches to the narratable prose form.
