// Package graphql renders an errx error as a GraphQL error: the
// end-user-safe message becomes Message, machine identity and safe fields
// go under Extensions (conventional extensions.code), and internal detail
// is never exposed. It consumes the neutral errx.Report.
//
// Wire it as a gqlgen ErrorPresenter:
//
//	srv.SetErrorPresenter(func(ctx context.Context, e error) *gqlerror.Error {
//		return graphqlx.Present(ctx, e)
//	})
package graphql

import (
	"context"

	"github.com/ubgo/errx"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// Present converts err into a *gqlerror.Error. If err carries no
// *errx.Error it falls back to a generic wrapped gqlerror so the resolver
// chain still gets a well-formed error.
func Present(ctx context.Context, err error) *gqlerror.Error {
	rep, ok := errx.Snapshot(ctx, err)
	if !ok {
		return gqlerror.WrapPath(nil, err)
	}
	msg := rep.Public
	if msg == "" {
		msg = "Internal server error"
	}
	ext := map[string]interface{}{}
	if rep.Code != "" {
		ext["code"] = rep.Code
	}
	if rep.Domain != "" {
		ext["domain"] = rep.Domain
	}
	ext["class"] = rep.Class.String()
	if rep.Fingerprint != "" {
		ext["fingerprint"] = rep.Fingerprint
	}
	if rep.Retryable {
		ext["retryable"] = true
		if rep.RetryAfter != "" {
			ext["retryAfter"] = rep.RetryAfter
		}
	}
	if rep.TraceID != "" {
		ext["traceId"] = rep.TraceID
	}
	if rep.DocURL != "" {
		ext["docUrl"] = rep.DocURL
	}
	if len(rep.Localized) > 0 {
		ext["localized"] = rep.Localized
	}
	fields := map[string]any{}
	for _, f := range rep.Fields {
		if f.Safe { // Snapshot already redacted unsafe values
			fields[f.Key] = f.Value
		}
	}
	if len(fields) > 0 {
		ext["fields"] = fields
	}
	return &gqlerror.Error{
		Message:    msg,
		Extensions: ext,
	}
}
