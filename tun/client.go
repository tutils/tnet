package tun

import "github.com/gorilla/websocket"

type Client interface {
	DialAndServe() error
}

type wsClient struct {
	opts *ClientOptions
	conn *websocket.Conn
}

func (c *wsClient) DialAndServe() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.opts.addr.String(), nil)
	if err != nil {
		return err
	}
	c.conn = conn
	if h := c.opts.handler; h != nil {
		wsr := newWsReader(conn)
		wsw := newWsWriter(conn)
		h.ServeTun(wsr, wsw)
	}
	return err
}

func NewClient(opts ...ClientOption) Client {
	opt := &ClientOptions{
		addr: DefaultConnectAddress,
	}
	for _, o := range opts {
		o(opt)
	}

	c := &wsClient{
		opts: opt,
	}
	return c
}

var DefaultConnectAddress = NewWsAddr("ws://127.0.0.1:8080/stream")

type ClientOptions struct {
	addr    Addr
	handler Handler
}

type ClientOption func(*ClientOptions)

func WithConnectAddress(addr Addr) ClientOption {
	return func(opts *ClientOptions) {
		opts.addr = addr
	}
}

func WithClientHandler(h Handler) ClientOption {
	return func(opts *ClientOptions) {
		opts.handler = h
	}
}
