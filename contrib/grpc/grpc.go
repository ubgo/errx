// Package grpc maps an errx error to a google.rpc.Status: a canonical gRPC
// code plus typed error details (ErrorInfo with the stable (reason,domain)
// identity, RetryInfo, LocalizedMessage, Help). It consumes the neutral
// errx.Report so unsafe fields never cross the wire.
package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/ubgo/errx"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/protoadapt"
	"google.golang.org/protobuf/types/known/durationpb"
)

// codeFor maps an errx code (then class) to a canonical gRPC code.
var codeFor = map[string]codes.Code{
	"NOT_FOUND":           codes.NotFound,
	"ALREADY_EXISTS":      codes.AlreadyExists,
	"CONFLICT":            codes.Aborted,
	"VALIDATION":          codes.InvalidArgument,
	"INVALID_ARGUMENT":    codes.InvalidArgument,
	"UNAUTHENTICATED":     codes.Unauthenticated,
	"PERMISSION_DENIED":   codes.PermissionDenied,
	"FORBIDDEN":           codes.PermissionDenied,
	"RESOURCE_EXHAUSTED":  codes.ResourceExhausted,
	"FAILED_PRECONDITION": codes.FailedPrecondition,
	"UNAVAILABLE":         codes.Unavailable,
	"UNIMPLEMENTED":       codes.Unimplemented,
	"TIMEOUT":             codes.DeadlineExceeded,
}

// CodeFor resolves the gRPC code for err.
func CodeFor(err error) codes.Code {
	rep, ok := errx.Snapshot(context.Background(), err)
	if !ok {
		return codes.Unknown
	}
	if c, found := codeFor[rep.Code]; found {
		return c
	}
	switch rep.Class {
	case errx.ClassDefect:
		return codes.Internal
	case errx.ClassCancelled:
		return codes.Canceled
	default:
		if rep.Retryable {
			return codes.Unavailable
		}
		return codes.InvalidArgument
	}
}

// Status builds a *status.Status with typed details. The end-user-safe
// message becomes the status message; ErrorInfo carries the stable
// (reason=code, domain) identity plus safe fields as metadata.
func Status(ctx context.Context, err error) *status.Status {
	rep, ok := errx.Snapshot(ctx, err)
	if !ok {
		return status.New(codes.Unknown, "unknown error")
	}
	msg := rep.Public
	if msg == "" {
		msg = rep.Message
	}
	st := status.New(CodeFor(err), msg)

	var dets []protoadapt.MessageV1
	if rep.Code != "" || rep.Domain != "" {
		meta := map[string]string{}
		for _, f := range rep.Fields {
			if f.Safe {
				meta[f.Key] = fmt.Sprintf("%v", f.Value)
			}
		}
		dets = append(dets, &errdetails.ErrorInfo{
			Reason:   rep.Code,
			Domain:   rep.Domain,
			Metadata: meta,
		})
	}
	if rep.Retryable {
		d := time.Duration(0)
		if rep.RetryAfter != "" {
			if pd, e := time.ParseDuration(rep.RetryAfter); e == nil {
				d = pd
			}
		}
		dets = append(dets, &errdetails.RetryInfo{RetryDelay: durationpb.New(d)})
	}
	if len(rep.Localized) > 0 {
		for locale, m := range rep.Localized {
			dets = append(dets, &errdetails.LocalizedMessage{Locale: locale, Message: m})
		}
	} else if rep.Public != "" {
		dets = append(dets, &errdetails.LocalizedMessage{Locale: "en-US", Message: rep.Public})
	}
	if rep.Hint != "" || rep.DocURL != "" {
		link := &errdetails.Help_Link{Description: rep.Hint}
		if rep.DocURL != "" {
			link.Url = rep.DocURL
		}
		dets = append(dets, &errdetails.Help{Links: []*errdetails.Help_Link{link}})
	}
	if len(dets) == 0 {
		return st
	}
	if withDetails, e := st.WithDetails(dets...); e == nil {
		return withDetails
	}
	return st
}

// Err is Status(...).Err() — ready to return from a gRPC handler. Returns
// nil if err is nil.
func Err(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	return Status(ctx, err).Err()
}
