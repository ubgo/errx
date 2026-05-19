package errx_test

import (
	"errors"
	"fmt"

	"github.com/ubgo/errx"
)

func Example() {
	err := errx.New("db: deadlock on user_orders").
		WithDomain("billing").
		WithCode("TX_RETRY").
		WithPublic("Something went wrong, please retry").
		WithRetryable(0)

	fmt.Println(err.Error())
	fmt.Println(err.Public("unknown"))
	fmt.Println(errx.Code(err))
	fmt.Println(errx.IsRetryable(err))
	// Output:
	// db: deadlock on user_orders
	// Something went wrong, please retry
	// TX_RETRY
	// true
}

func ExampleWrap() {
	sentinel := errors.New("pq: deadlock")
	err := errx.Wrap(sentinel, "load orders").WithCode("DB_READ")

	fmt.Println(err.Error())
	fmt.Println(errors.Is(err, sentinel)) // transparent by default
	fmt.Println(errx.Code(err))
	// Output:
	// load orders: pq: deadlock
	// true
	// DB_READ
}

func ExampleError_Opaque() {
	sentinel := errors.New("secret dependency sentinel")
	err := errx.Wrap(sentinel, "public boundary").Opaque()

	fmt.Println(errors.Is(err, sentinel)) // barrier hides the cause
	// Output:
	// false
}

func ExampleAccumulate() {
	err := errx.Accumulate(
		func() error { return nil },
		func() error { return errors.New("name is required") },
		func() error { return errors.New("email is invalid") },
	)
	fmt.Println(errx.Code(err))
	fmt.Println(len(errx.Get(err).Suppressed()))
	// Output:
	// VALIDATION
	// 2
}

func ExampleNote() {
	base := errx.New("upstream timeout").WithCode("TIMEOUT")
	_ = errx.Note(base, "attempt", 3) // no extra wrapper frame
	for _, f := range base.Fields() {
		fmt.Printf("%s=%v\n", f.Key, f.Value)
	}
	// Output:
	// attempt=3
}

func ExampleEncode() {
	orig := errx.New("internal detail").
		WithDomain("billing").WithCode("NOT_FOUND").
		WithPublic("Order not found").
		With("password", "hunter2") // unsafe: never serialized

	blob, _ := errx.Encode(orig)
	got, _ := errx.Decode(blob)

	fmt.Println(errx.Code(got))
	fmt.Println(got.Public("x"))
	fmt.Println(got.Fingerprint() == orig.Fingerprint())
	// Output:
	// NOT_FOUND
	// Order not found
	// true
}
