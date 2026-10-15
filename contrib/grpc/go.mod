module github.com/ubgo/errx/contrib/grpc

go 1.25.0

require (
	github.com/ubgo/errx v0.0.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa
	google.golang.org/grpc v1.83.0
	google.golang.org/protobuf v1.36.11
)

require golang.org/x/sys v0.45.0 // indirect

replace github.com/ubgo/errx => ../..
