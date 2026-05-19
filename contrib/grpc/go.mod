module github.com/ubgo/errx/contrib/grpc

go 1.24

require (
	github.com/ubgo/errx v0.0.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241104194629-dd2ea8efbc28
	google.golang.org/grpc v1.68.0
	google.golang.org/protobuf v1.35.1
)

require golang.org/x/sys v0.25.0 // indirect

replace github.com/ubgo/errx => ../..
