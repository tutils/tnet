package tun

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
)

type wsClient struct {
	opts ClientOptions
	conn *websocket.Conn
}

func (c *wsClient) Handler() Handler {
	return c.opts.handler
}

func (c *wsClient) DialAndServe() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.opts.addr, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	c.conn = conn
	if h := c.opts.handler; h != nil {
		wsr := newWsReader(conn)
		wsw := newWsWriter(conn)
		ctx := context.Background()

		conn.SetReadDeadline(time.Now().Add(readTimeout))
		origPingHandler := conn.PingHandler()
		conn.SetPingHandler(func(appData string) error {
			conn.SetReadDeadline(time.Now().Add(readTimeout))
			return origPingHandler(appData)
		})

		h.ServeTun(ctx, wsr, wsw)
	}
	return nil
}

func newWsClient(opts ...ClientOption) Client {
	opt := newClientOptions(opts...)

	c := &wsClient{
		opts: *opt,
	}
	return c
}
