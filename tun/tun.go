package tun

import "io"

// tunnel address
type Addr interface {
	String() string
}

// tunnel handler
type Handler interface {
	ServeTun(r io.Reader, w io.Writer)
}
