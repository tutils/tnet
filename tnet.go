package tnet

import (
	"io"
	"sync"
)

// SyncReader is concurrency safe reader
type SyncReader struct {
	r  io.Reader
	mu sync.Mutex
}

func (r *SyncReader) Read(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.r.Read(p)
}

// NewSyncReader create a new SyncReader
func NewSyncReader(r io.Reader) io.Reader {
	return &SyncReader{r: r}
}

// SyncWriter is concurrency safe writer
type SyncWriter struct {
	w  io.Writer
	mu sync.Mutex
}

func (w *SyncWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.Write(p)
}

// NewSyncWriter create a new SyncWriter
func NewSyncWriter(w io.Writer) io.Writer {
	return &SyncWriter{w: w}
}
