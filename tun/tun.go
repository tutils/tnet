package tun

import "io"

type Addr interface {
	String() string
}

type Handler interface {
	ServeTun(r io.Reader, w io.Writer)
}
