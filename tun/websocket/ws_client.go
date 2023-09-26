package websocket

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tutils/tnet/tun"
)

var _ tun.Client = &client{}

type client struct {
	opts tun.ClientOptions
	conn *websocket.Conn
}

func (c *client) Handler() tun.Handler {
	return c.opts.Handler
}

func (c *client) DialAndServe() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.opts.Address, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	c.conn = conn
	if h := c.Handler(); h != nil {
		tunr := newReader(conn)
		tunw := newWriter(conn)
		ctx := context.Background()

		conn.SetReadDeadline(time.Now().Add(readTimeout))
		origPingHandler := conn.PingHandler()
		conn.SetPingHandler(func(appData string) error {
			conn.SetReadDeadline(time.Now().Add(readTimeout))
			return origPingHandler(appData)
		})

		h.ServeTun(ctx, tunr, tunw)
	}
	return nil
}

func NewClient(opts ...tun.ClientOption) tun.Client {
	opt := tun.NewClientOptions(opts...)

	c := &client{
		opts: *opt,
	}
	return c
}
