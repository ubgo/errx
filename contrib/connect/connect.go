// Package connect maps an errx error to a *connect.Error (ConnectRPC):
// canonical code from the stable identity/class plus an ErrorInfo detail
// carrying (reason, domain) and safe metadata. Consumes the neutral
// errx.Report, so unsafe fields never cross the wire.
package connect

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/ubgo/errx"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

var codeFor = map[string]connect.Code{
	"NOT_FOUND":           connect.CodeNotFound,
	"ALREADY_EXISTS":      connect.CodeAlreadyExists,
	"CONFLICT":            connect.CodeAborted,
	"VALIDATION":          connect.CodeInvalidArgument,
	"INVALID_ARGUMENT":    connect.CodeInvalidArgument,
	"UNAUTHENTICATED":     connect.CodeUnauthenticated,
	"PERMISSION_DENIED":   connect.CodePermissionDenied,
	"FORBIDDEN":           connect.CodePermissionDenied,
	"RESOURCE_EXHAUSTED":  connect.CodeResourceExhausted,
	"FAILED_PRECONDITION": connect.CodeFailedPrecondition,
	"UNAVAILABLE":         connect.CodeUnavailable,
	"UNIMPLEMENTED":       connect.CodeUnimplemented,
	"TIMEOUT":             connect.CodeDeadlineExceeded,
}

// CodeFor resolves the Connect code for err.
func CodeFor(err error) connect.Code {
	rep, ok := errx.Snapshot(context.Background(), err)
	if !ok {
		return connect.CodeUnknown
	}
	if c, found := codeFor[rep.Code]; found {
		return c
	}
	switch rep.Class {
	case errx.ClassDefect:
		return connect.CodeInternal
	case errx.ClassCancelled:
		return connect.CodeCanceled
	default:
		if rep.Retryable {
			return connect.CodeUnavailable
		}
		return connect.CodeInvalidArgument
	}
}

// Error converts err into a *connect.Error. The end-user-safe message is
// the surfaced message; an ErrorInfo detail carries the stable
// (reason=code, domain) identity plus safe fields.
func Error(ctx context.Context, err error) *connect.Error {
	rep, ok := errx.Snapshot(ctx, err)
	if !ok {
		return connect.NewError(connect.CodeUnknown, err)
	}
	msg := rep.Public
	if msg == "" {
		msg = rep.Message
	}
	ce := connect.NewError(CodeFor(err), fmt.Errorf("%s", msg))

	if rep.Code != "" || rep.Domain != "" {
		meta := map[string]string{}
		for _, f := range rep.Fields {
			if f.Safe {
				meta[f.Key] = fmt.Sprintf("%v", f.Value)
			}
		}
		if d, e := connect.NewErrorDetail(&errdetails.ErrorInfo{
			Reason:   rep.Code,
			Domain:   rep.Domain,
			Metadata: meta,
		}); e == nil {
			ce.AddDetail(d)
		}
	}
	return ce
}
