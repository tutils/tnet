package common

import (
	"io"
)

func Copy(tunw io.Writer, r io.Reader, packFunc func(tunw io.Writer, data []byte) error) error {
	var err error
	var buf []byte
	size := 32 * 1024
	buf = make([]byte, size)
	for {
		nr, er := r.Read(buf)
		if nr > 0 {
			if ew := packFunc(tunw, buf[:nr]); ew != nil {
				err = ew
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return err
}
