package proxy

import (
	"fmt"
	"io"
	"math"

	"github.com/tutils/tnet/counter"
)

type counterWriter struct {
	w io.Writer
	c counter.Counter
}

func (w *counterWriter) Write(p []byte) (n int, err error) {
	n, err = w.w.Write(p)
	if err == nil && n >= 0 {
		w.c.Add(int64(n))
	}
	return n, err
}

type counterReader struct {
	r io.Reader
	c counter.Counter
}

func (r *counterReader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	if err == nil && n >= 0 {
		r.c.Add(int64(n))
	}
	return n, err
}

var units = []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
var log1024 = math.Log(1024.0)

func humanReadable(bytes uint64) string {
	base := 1024.0
	f := float64(bytes)
	if f < base {
		return fmt.Sprintf("%d %s", bytes, units[0])
	}
	exp := int(math.Log(f) / log1024)
	roundedSize := int64(f / math.Pow(base, float64(exp)))
	return fmt.Sprintf("%d %s", roundedSize, units[exp])
}
