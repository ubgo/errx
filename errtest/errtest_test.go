package errtest_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ubgo/errx"
	"github.com/ubgo/errx/errtest"
)

// stub captures Fatalf instead of failing, so we can assert both the
// pass and the fail behavior of every helper.
type stub struct {
	failed bool
	msg    string
}

func (s *stub) Helper() {}
func (s *stub) Fatalf(format string, args ...any) {
	s.failed = true
	s.msg = fmt.Sprintf(format, args...)
}

func TestHelpersPass(t *testing.T) {
	e := errx.New("x").
		WithCode("E_X").
		WithClass(errx.ClassDefect).
		WithRetryable(time.Second)
	s := &stub{}

	errtest.Code(s, e, "E_X")
	errtest.Class(s, e, errx.ClassDefect)
	errtest.Retryable(s, e, true)
	errtest.HasCode(s, errx.Wrap(e, "ctx"), "E_X")
	errtest.NoError(s, nil)

	mk := func(m string) error { return errx.New(m).WithDomain("d").WithCode("C") }
	errtest.Fingerprint(s, mk("a 1"), mk("a 2"))

	if s.failed {
		t.Fatalf("helpers should pass, got: %s", s.msg)
	}
}

func TestInChainReal(t *testing.T) {
	sentinel := errors.New("sentinel")
	s := &stub{}
	errtest.InChain(s, errx.Wrap(sentinel, "wrap"), sentinel)
	if s.failed {
		t.Fatalf("InChain should pass for a wrapped sentinel: %s", s.msg)
	}
}

func TestHelpersFail(t *testing.T) {
	e := errx.New("x").WithCode("A").WithClass(errx.ClassExpected)

	cases := []struct {
		name string
		run  func(t errtest.TB)
		want string
	}{
		{"Code", func(tb errtest.TB) { errtest.Code(tb, e, "B") }, "errx.Code"},
		{"Class", func(tb errtest.TB) { errtest.Class(tb, e, errx.ClassDefect) }, "ClassOf"},
		{"Retryable", func(tb errtest.TB) { errtest.Retryable(tb, e, true) }, "IsRetryable"},
		{"InChain", func(tb errtest.TB) { errtest.InChain(tb, e, errors.New("nope")) }, "errors.Is"},
		{"HasCode", func(tb errtest.TB) { errtest.HasCode(tb, e, "ZZZ") }, "expected code"},
		{"NoError", func(tb errtest.TB) { errtest.NoError(tb, errors.New("boom")) }, "unexpected error"},
		{"Fingerprint", func(tb errtest.TB) {
			errtest.Fingerprint(tb, errx.New("p").WithCode("X"), errors.New("plain"))
		}, "fingerprints differ"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &stub{}
			c.run(s)
			if !s.failed {
				t.Fatalf("%s should have failed", c.name)
			}
			if !strings.Contains(s.msg, c.want) {
				t.Fatalf("%s message = %q, want substring %q", c.name, s.msg, c.want)
			}
		})
	}
}
