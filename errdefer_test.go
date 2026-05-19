package errx_test

import (
	"errors"
	"testing"

	"github.com/ubgo/errx"
)

func TestOnErrorRunsOnlyOnFailure(t *testing.T) {
	rolledBack := false
	fail := func() (err error) {
		defer errx.OnError(&err, func() { rolledBack = true })
		return errors.New("boom")
	}
	_ = fail()
	if !rolledBack {
		t.Fatal("OnError should run cleanup on the failure path")
	}

	rolledBack = false
	ok := func() (err error) {
		defer errx.OnError(&err, func() { rolledBack = true })
		return nil
	}
	_ = ok()
	if rolledBack {
		t.Fatal("OnError must NOT run cleanup on success")
	}
}

func TestOnSuccess(t *testing.T) {
	committed := false
	run := func() (err error) {
		defer errx.OnSuccess(&err, func() { committed = true })
		return nil
	}
	_ = run()
	if !committed {
		t.Fatal("OnSuccess should run on the success path")
	}
}

type closer struct{ err error }

func (c closer) Close() error { return c.err }

func TestCloseSuppressingDoesNotMaskPrimary(t *testing.T) {
	primary := errors.New("primary failure")
	closeErr := errors.New("close failed")

	op := func() (err error) {
		err = errx.New("op failed").WithCode("OP")
		defer errx.CloseSuppressing(&err, closer{err: closeErr})
		_ = primary
		return err
	}
	got := op()
	if errx.Code(got) != "OP" {
		t.Fatalf("primary identity lost: %q", errx.Code(got))
	}
	if !errors.Is(got, closeErr) {
		t.Fatal("close error should be attached (suppressed) and still matchable")
	}
	e := errx.Get(got)
	if e == nil || len(e.Suppressed()) != 1 {
		t.Fatalf("close error not suppressed: %+v", e)
	}
}

func TestCloseSuppressingBecomesErrorWhenNoPrimary(t *testing.T) {
	closeErr := errors.New("close failed")
	op := func() (err error) {
		defer errx.CloseSuppressing(&err, closer{err: closeErr})
		return nil
	}
	if !errors.Is(op(), closeErr) {
		t.Fatal("with no primary, the close error must surface")
	}
}
