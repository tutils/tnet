package crypt

import (
	"io"
)

type Crypt interface {
	NewEncoder(w io.Writer) io.Writer
	NewDecoder(r io.Reader) io.Reader
}
