package tun

import "github.com/gorilla/websocket"

type Client interface {
	DialAndServe() error
}

type wsClient struct {
	opts ClientOptions
	conn *websocket.Conn
}

func (c *wsClient) DialAndServe() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.opts.addr, nil)
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
	opt := newClientOptions(opts...)

	c := &wsClient{
		opts: *opt,
	}
	return c
}
