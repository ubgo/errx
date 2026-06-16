module github.com/ubgo/errx/contrib/sentry

go 1.25.0

require (
	github.com/getsentry/sentry-go v0.47.0
	github.com/ubgo/errx v0.0.0
)

require (
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/text v0.37.0 // indirect
)

replace github.com/ubgo/errx => ../..
