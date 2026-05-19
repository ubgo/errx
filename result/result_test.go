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
