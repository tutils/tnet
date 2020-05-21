package crypt

import (
	"io"
)

// Crypt wrap reader and writer
type Crypt interface {
	NewEncoder(w io.Writer) io.Writer
	NewDecoder(r io.Reader) io.Reader
}
