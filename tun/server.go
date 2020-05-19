package tun

import (
	"bytes"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
)

type Addr interface {
	String() string
}

type Server interface {
	ListenAndServe() error
}

type Handler interface {
	ServeTun(r io.Reader, w io.Writer)
}

type wsAddr struct {
	url *url.URL
}

func (a *wsAddr) String() string {
	return a.url.String()
}

func (a *wsAddr) Host() string {
	return a.url.Host
}

func (a *wsAddr) URI() string {
	return a.url.RequestURI()
}

func NewWsAddr(rawUrl string) Addr {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil
	}

	return &wsAddr{
		url: u,
	}
}

type wsServer struct {
	opts *ServerOptions
	srv  *http.Server
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

func (s *wsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	if h := s.opts.handler; h != nil {
		wsr := newWsReader(conn)
		wsw := newWsWriter(conn)
		h.ServeTun(wsr, wsw)
	}
}

var eofReader = bytes.NewReader(nil)

type wsReader struct {
	conn *websocket.Conn
	r    io.Reader
}

func (wsr *wsReader) Read(p []byte) (n int, err error) {
	n, err = wsr.r.Read(p)
	if err == io.EOF {
		_, wsr.r, err = wsr.conn.NextReader()
		if err != nil {
			return 0, err
		}
		return wsr.r.Read(p)
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

func (s *wsServer) ListenAndServe() error {
	return s.srv.ListenAndServe()
}

func NewServer(opts ...ServerOption) Server {
	opt := &ServerOptions{
		addr: DefaultListenAddress,
	}
	for _, o := range opts {
		o(opt)
	}

	s := &wsServer{
		opts: opt,
	}

	addr := opt.addr.(*wsAddr)
	mux := http.NewServeMux()
	mux.Handle(addr.URI(), s)
	srv := &http.Server{
		Addr:    addr.Host(),
		Handler: mux,
	}
	s.srv = srv

	return s
}

var DefaultListenAddress = NewWsAddr("ws://0.0.0.0:8080/stream")

type ServerOptions struct {
	addr    Addr
	handler Handler
}

type ServerOption func(*ServerOptions)

func WithListenAddress(addr Addr) ServerOption {
	return func(opts *ServerOptions) {
		opts.addr = addr
	}
}

func WithServerHandler(h Handler) ServerOption {
	return func(opts *ServerOptions) {
		opts.handler = h
	}
}
