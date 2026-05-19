module github.com/ubgo/errx/contrib/connect

go 1.24.0

require (
	connectrpc.com/connect v1.19.2
	github.com/ubgo/errx v0.0.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241104194629-dd2ea8efbc28
)

require google.golang.org/protobuf v1.36.9 // indirect

replace github.com/ubgo/errx => ../..
