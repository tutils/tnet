package xor

import (
	"io"
	"math/rand"

	"github.com/tutils/tnet/crypt"
)

var _ crypt.Crypt = &xorCrypt{}

type xorCrypt struct {
	seed int64
}

func (c *xorCrypt) NewEncoder(w io.Writer) io.Writer {
	return &xorEncoder{
		w:   w,
		rnd: rand.New(rand.NewSource(c.seed)),
	}
}

func (c *xorCrypt) NewDecoder(r io.Reader) io.Reader {
	return &xorDecoder{
		r:   r,
		rnd: rand.New(rand.NewSource(c.seed)),
	}
}

// NewCrypt create a new Crypt
func NewCrypt(seed int64) crypt.Crypt {
	return &xorCrypt{
		seed: seed,
	}
}

type xorEncoder struct {
	w   io.Writer
	rnd *rand.Rand
	buf []byte
}

func (e *xorEncoder) Write(p []byte) (n int, err error) {
	n = len(p)
	if cap(e.buf) < n {
		e.buf = make([]byte, n)
	} else {
		e.buf = e.buf[:n]
	}

	e.rnd.Read(e.buf)
	for i, b := range p {
		e.buf[i] ^= b
	}

	return e.w.Write(e.buf)
}

type xorDecoder struct {
	r   io.Reader
	rnd *rand.Rand
	buf []byte
}

func (d *xorDecoder) Read(p []byte) (n int, err error) {
	n, err = d.r.Read(p)
	if err != nil {
		return n, err
	}
	if cap(d.buf) < n {
		d.buf = make([]byte, n)
	} else {
		d.buf = d.buf[:n]
	}

	d.rnd.Read(d.buf)
	for i, b := range d.buf {
		p[i] ^= b
	}

	return n, nil
}
