module github.com/ubgo/errx/contrib/otel

go 1.24

require (
	github.com/ubgo/errx v0.0.0
	go.opentelemetry.io/otel v1.32.0
	go.opentelemetry.io/otel/trace v1.32.0
)

replace github.com/ubgo/errx => ../..
