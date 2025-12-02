package tun

import (
	"context"

	"github.com/gorilla/websocket"
)

var _ Client = &wsClient{}

type wsClient struct {
	opts ClientOptions
}

func (c *wsClient) DialAndServe(ctx context.Context, h Handler) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.opts.addr, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	wsr := newWsReader(conn)
	wsw := newWsWriter(conn)

	done := make(chan struct{})
	go startPing(conn, done)
	h.ServeTun(ctx, wsr, wsw)

	close(done)
	return nil
}

func newWsClient(opts ...ClientOption) Client {
	opt := newClientOptions(opts...)

	c := &wsClient{
		opts: *opt,
	}
	return c
}
