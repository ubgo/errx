module github.com/ubgo/errx/contrib/connect

go 1.24

require (
	connectrpc.com/connect v1.17.0
	github.com/ubgo/errx v0.0.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241104194629-dd2ea8efbc28
)

require google.golang.org/protobuf v1.35.1 // indirect

replace github.com/ubgo/errx => ../..
