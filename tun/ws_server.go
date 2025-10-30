package tun

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type wsAddr struct {
	url *url.URL
}

func (a *wsAddr) String() string {
	return a.url.String()
}

func (a *wsAddr) host() string {
	return a.url.Host
}

func (a *wsAddr) uri() string {
	return a.url.RequestURI()
}

func newWsAddr(rawURL string) *wsAddr {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}

	return &wsAddr{
		url: u,
	}
}

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  4 << 10,
		WriteBufferSize: 4 << 10,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

var eofReader = bytes.NewReader(nil)

type wsReader struct {
	conn *websocket.Conn
	r    io.Reader
}

func (wsr *wsReader) Read(p []byte) (n int, err error) {
	defer func() {
		if n > 0 {
			updateReadDeadline(wsr.conn)
		}
	}()
	n, err = wsr.r.Read(p)
	if err == io.EOF {
		for {
			var typ int
			typ, wsr.r, err = wsr.conn.NextReader()
			if err != nil {
				return 0, err
			}
			if typ == websocket.BinaryMessage {
				return wsr.r.Read(p)
			}
			if _, err := io.ReadAll(wsr.r); err != nil {
				return 0, err
			}
		}
	}
	return n, err
}

func newWsReader(conn *websocket.Conn) io.Reader {
	return &wsReader{
		conn: conn,
		r:    eofReader,
	}
}

type wsWriter struct {
	conn *websocket.Conn
}

func (wsw *wsWriter) Write(p []byte) (n int, err error) {
	err = wsw.conn.WriteMessage(websocket.BinaryMessage, p)
	return len(p), err
}

func newWsWriter(conn *websocket.Conn) io.Writer {
	return &wsWriter{
		conn: conn,
	}
}

var _ Server = &wsServer{}

type wsServer struct {
	opts ServerOptions
}

func (s *wsServer) serveHTTP(h Handler, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	wsr := newWsReader(conn)
	wsw := newWsWriter(conn)
	ctx := r.Context()

	done := make(chan struct{})
	go startPing(conn, done)
	h.ServeTun(ctx, wsr, wsw)

	close(done)
}

func (s *wsServer) ListenAndServe(h Handler) error {
	addr := newWsAddr(s.opts.addr)
	if addr == nil {
		return errors.New("invalid address")
	}
	mux := http.NewServeMux()
	mux.HandleFunc(addr.uri(), func(w http.ResponseWriter, r *http.Request) {
		s.serveHTTP(h, w, r)
	})
	var connID int64
	srv := &http.Server{
		Addr:    addr.host(),
		Handler: mux,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			connID++
			return context.WithValue(ctx, ConnIDContextKey{}, connID)
		},
	}
	return srv.ListenAndServe()
}

// ConnIDContextKey is context key of connID
type ConnIDContextKey struct{}

func newWsServer(opts ...ServerOption) Server {
	opt := newServerOptions(opts...)

	s := &wsServer{
		opts: *opt,
	}
	return s
}

const readTimeout = time.Second * 15
const pingPeriod = time.Second * 10
const writeTimeout = time.Second * 5

func updateReadDeadline(conn *websocket.Conn) {
	conn.SetReadDeadline(time.Now().Add(readTimeout))
}

func startPing(conn *websocket.Conn, done chan struct{}) {
	conn.SetReadDeadline(time.Now().Add(readTimeout))
	conn.SetPongHandler(func(string) error {
		updateReadDeadline(conn)
		return nil
	})
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// log.Println("@@send ping")
			conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeTimeout))
		case <-done:
			return
		}
	}
}
