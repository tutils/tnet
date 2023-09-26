package websocket

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tutils/tnet/tun"
)

type addr struct {
	url *url.URL
}

func (a *addr) String() string {
	return a.url.String()
}

func (a *addr) host() string {
	return a.url.Host
}

func (a *addr) uri() string {
	return a.url.RequestURI()
}

func newAddr(rawURL string) *addr {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}

	return &addr{
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

type reader struct {
	conn *websocket.Conn
	r    io.Reader
}

func (r *reader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	if err == io.EOF {
		for {
			var typ int
			typ, r.r, err = r.conn.NextReader()
			if err != nil {
				return 0, err
			}
			if typ == websocket.BinaryMessage {
				return r.r.Read(p)
			}
			if _, err := io.ReadAll(r.r); err != nil {
				return 0, err
			}
		}
	}
	return n, err
}

func newReader(conn *websocket.Conn) *reader {
	return &reader{
		conn: conn,
		r:    eofReader,
	}
}

type writer struct {
	conn *websocket.Conn
}

func (w *writer) Write(p []byte) (n int, err error) {
	err = w.conn.WriteMessage(websocket.BinaryMessage, p)
	return len(p), err
}

func newWriter(conn *websocket.Conn) *writer {
	return &writer{
		conn: conn,
	}
}

var _ tun.Server = &server{}

type server struct {
	opts tun.ServerOptions
	srv  *http.Server
}

func (s *server) Handler() tun.Handler {
	return s.opts.Handler
}

// ServeHTTP implements http.Handler
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	if h := s.Handler(); h != nil {
		tunr := newReader(conn)
		tunw := newWriter(conn)
		ctx := r.Context()

		done := make(chan struct{})
		go startPing(conn, done)

		h.ServeTun(ctx, tunr, tunw)

		close(done)
	}
}

func (s *server) ListenAndServe() error {
	return s.srv.ListenAndServe()
}

func NewServer(opts ...tun.ServerOption) tun.Server {
	opt := tun.NewServerOptions(opts...)

	s := &server{
		opts: *opt,
	}

	addr := newAddr(opt.Address)
	mux := http.NewServeMux()
	mux.Handle(addr.uri(), s)
	var tunID int64
	srv := &http.Server{
		Addr:    addr.host(),
		Handler: mux,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			tunID++
			return context.WithValue(ctx, tun.TunIDContextKey{}, tunID)
		},
	}
	s.srv = srv

	return s
}

const readTimeout = time.Second * 15
const pingPeriod = time.Second * 10
const writeTimeout = time.Second

func startPing(conn *websocket.Conn, done chan struct{}) {
	conn.SetReadDeadline(time.Now().Add(readTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeTimeout))
		case <-done:
			return
		}
	}
}
