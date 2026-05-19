module github.com/ubgo/errx/contrib/otel

go 1.25.0

require (
	github.com/ubgo/errx v0.0.0
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/trace v1.43.0
)

require github.com/cespare/xxhash/v2 v2.3.0 // indirect

replace github.com/ubgo/errx => ../..
