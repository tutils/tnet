package tun

import (
	"io"
	"testing"
	"time"
)

type chandler struct {
	t *testing.T
}

func (h *chandler) ServeTun(r io.Reader, w io.Writer) {
	buf := make([]byte, 1<<10)
	w.Write([]byte("Hello"))
	r.Read(buf)
	h.t.Log(string(buf))

	time.Sleep(time.Second)

	w.Write([]byte("I'm t5w0rd"))
	r.Read(buf)
	h.t.Log(string(buf))
}

func TestNewClient(t *testing.T) {
	h := &chandler{t: t}
	c := NewClient(
		WithClientHandler(h),
	)
	if err := c.DialAndServe(); err != nil {
		t.Fatal(err)
	}
}
