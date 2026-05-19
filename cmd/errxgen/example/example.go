// Package example demonstrates errxgen. Run `go generate ./...` to (re)emit
// example_errxgen.go from the directives below.
package example

//go:generate go run github.com/ubgo/errx/cmd/errxgen .

// errxgen: message="open %s: permission denied (uid %d)" args=Path,UID code=IO_DENIED unwrap=Err
type DeniedError struct {
	Path string
	UID  int
	Err  error
}

// errxgen: message="config key %q is required" args=Key code=CONFIG_MISSING
type MissingKeyError struct {
	Key string
}
