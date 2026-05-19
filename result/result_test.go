package result_test

import (
	"errors"
	"strconv"
	"testing"

	"github.com/ubgo/errx"
	"github.com/ubgo/errx/result"
)

func TestOkErrAndAccessors(t *testing.T) {
	ok := result.Ok(42)
	if !ok.IsOk() || ok.IsErr() {
		t.Fatal("Ok state wrong")
	}
	if v, good := ok.Value(); !good || v != 42 {
		t.Fatalf("Value = %v %v", v, good)
	}
	if ok.UnwrapOr(0) != 42 || ok.Unwrap() != 42 {
		t.Fatal("unwrap wrong")
	}

	e := result.Err[int](errx.New("bad").WithCode("X"))
	if e.IsOk() || !e.IsErr() {
		t.Fatal("Err state wrong")
	}
	if e.UnwrapOr(7) != 7 {
		t.Fatal("UnwrapOr should return default on error")
	}
	if errx.Code(e.Err()) != "X" {
		t.Fatalf("error not preserved: %v", e.Err())
	}
}

func TestUnwrapPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Unwrap on Err should panic")
		}
	}()
	_ = result.Err[int](errors.New("boom")).Unwrap()
}

func TestMapAndThenChaining(t *testing.T) {
	r := result.From(strconv.Atoi("21"))
	doubled := result.Map(r, func(n int) int { return n * 2 })
	if doubled.UnwrapOr(0) != 42 {
		t.Fatalf("Map result = %v", doubled.UnwrapOr(0))
	}

	parse := func(s string) result.Result[int] { return result.From(strconv.Atoi(s)) }
	chained := result.AndThen(result.Ok("10"), parse)
	if chained.UnwrapOr(-1) != 10 {
		t.Fatalf("AndThen = %v", chained.UnwrapOr(-1))
	}
	bad := result.AndThen(result.Ok("nope"), parse)
	if bad.IsOk() {
		t.Fatal("AndThen should propagate the parse error")
	}
}

func TestTryAndUnwrapOrElse(t *testing.T) {
	ok := result.Try(func() (int, error) { return 7, nil })
	if v, good := ok.Value(); !good || v != 7 {
		t.Fatalf("Try ok = %v %v", v, good)
	}
	bad := result.Try(func() (int, error) { return 0, errors.New("boom") })
	if bad.IsOk() {
		t.Fatal("Try should capture the error")
	}
	if got := bad.UnwrapOrElse(func(err error) int {
		if err == nil {
			t.Fatal("UnwrapOrElse should receive the error")
		}
		return 42
	}); got != 42 {
		t.Fatalf("UnwrapOrElse = %d", got)
	}
	if result.Ok(5).UnwrapOrElse(func(error) int { return -1 }) != 5 {
		t.Fatal("UnwrapOrElse on Ok should return the value")
	}
}

func TestErrorBranchesPropagate(t *testing.T) {
	e := result.Err[int](errors.New("orig"))

	if v := result.Map(e, func(n int) int { return n * 2 }); v.IsOk() {
		t.Fatal("Map must propagate error")
	}
	if v := result.AndThen(e, func(int) result.Result[int] { return result.Ok(1) }); v.IsOk() {
		t.Fatal("AndThen must propagate error")
	}
	if v := result.Match(e, func(int) string { return "ok" }, func(error) string { return "err" }); v != "err" {
		t.Fatalf("Match error branch = %q", v)
	}
	// MapErr / Recover no-op on Ok.
	if result.Ok(3).MapErr(func(err error) error { return err }).UnwrapOr(0) != 3 {
		t.Fatal("MapErr on Ok should be a no-op")
	}
	if result.Ok(3).Recover(func(error) int { return 9 }).UnwrapOr(0) != 3 {
		t.Fatal("Recover on Ok should be a no-op")
	}
}

func TestRecoverMapErrMatchCollect(t *testing.T) {
	rec := result.Err[int](errors.New("x")).Recover(func(error) int { return 99 })
	if rec.UnwrapOr(0) != 99 {
		t.Fatal("Recover failed")
	}
	me := result.Err[int](errors.New("low")).MapErr(func(e error) error {
		return errx.Wrap(e, "high")
	})
	if me.Err() == nil || me.Err().Error() != "high: low" {
		t.Fatalf("MapErr = %v", me.Err())
	}
	got := result.Match(result.Ok(5), func(n int) string { return "ok" + strconv.Itoa(n) }, func(error) string { return "err" })
	if got != "ok5" {
		t.Fatalf("Match = %q", got)
	}
	all := result.Collect(result.Ok(1), result.Ok(2), result.Ok(3))
	if vs, _ := all.Value(); len(vs) != 3 {
		t.Fatalf("Collect = %v", vs)
	}
	ff := result.Collect(result.Ok(1), result.Err[int](errors.New("stop")), result.Ok(3))
	if ff.IsOk() {
		t.Fatal("Collect must be fail-fast")
	}
}
