package example_test

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/ubgo/errx"
	"github.com/ubgo/errx/cmd/errxgen/example"
)

func TestGeneratedErrorAndCode(t *testing.T) {
	cause := fs.ErrPermission
	e := &example.DeniedError{Path: "/etc/secret", UID: 0, Err: cause}

	if got := e.Error(); got != "open /etc/secret: permission denied (uid 0)" {
		t.Fatalf("generated Error() = %q", got)
	}
	if e.Code() != "IO_DENIED" {
		t.Fatalf("generated Code() = %q", e.Code())
	}
	// errx.Code/HasCode pick up the generated Code() method even though
	// this is NOT an *errx.Error (behavioral detection).
	if errx.Code(e) != "IO_DENIED" || !errx.HasCode(e, "IO_DENIED") {
		t.Fatalf("errx.Code via method = %q", errx.Code(e))
	}
	// Generated Unwrap() makes the cause matchable.
	if !errors.Is(e, fs.ErrPermission) {
		t.Fatal("generated Unwrap() should expose the cause")
	}
	// Works through an errx.Wrap too.
	wrapped := errx.Wrap(e, "loading config")
	if errx.Code(wrapped) != "IO_DENIED" || !errors.Is(wrapped, fs.ErrPermission) {
		t.Fatalf("through errx.Wrap: code=%q", errx.Code(wrapped))
	}

	mk := &example.MissingKeyError{Key: "DATABASE_URL"}
	if mk.Error() != `config key "DATABASE_URL" is required` {
		t.Fatalf("MissingKeyError.Error() = %q", mk.Error())
	}
	var target *example.MissingKeyError
	if !errors.As(errx.Wrap(mk, "startup"), &target) {
		t.Fatal("errors.As should find the generated typed error")
	}
}
