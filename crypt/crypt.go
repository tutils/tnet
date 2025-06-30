package crypt

import (
	"io"
)

// Crypt wrap reader and writer
type Crypt interface {
	NewEncoder(w io.Writer, opts ...EncoderOption) io.Writer
	NewDecoder(r io.Reader, opts ...DecoderOption) io.Reader
}
