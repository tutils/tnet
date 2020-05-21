package tun

import (
	"context"
	"io"
)

// Addr is tunnel address
type Addr interface {
	String() string
}

// Handler is tunnel handler
type Handler interface {
	ServeTun(ctx context.Context, r io.Reader, w io.Writer)
}
