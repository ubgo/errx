module github.com/ubgo/errx/contrib/sentry

go 1.24

require (
	github.com/getsentry/sentry-go v0.31.1
	github.com/ubgo/errx v0.0.0
)

require (
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/ubgo/errx => ../..
