package tun

import (
	"io"
	"testing"
)

type shandler struct {
	t *testing.T
}

func (h *shandler) ServeTun(ctx context.Context, r io.Reader, w io.Writer) {
	buf := make([]byte, 1<<10)
	for {
		_, err := r.Read(buf)
		if err != nil {
			h.t.Log(err)
			return
		}
		h.t.Log(string(buf))
		_, err = w.Write([]byte("OK"))
		if err != nil {
			h.t.Log(err)
			return
		}
	}
}

func TestNewServer(t *testing.T) {
	h := &shandler{t: t}
	s := NewServer(
		WithServerHandler(h),
	)
	if err := s.ListenAndServe(); err != nil {
		t.Fatal(err)
	}
}
