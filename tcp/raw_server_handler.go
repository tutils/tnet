package tcp

import "context"

type RawTCPHandler interface {
	ServeTCP(ctx context.Context, conn *Conn)
}

type RawTCPServerHandler struct {
	Handler RawTCPHandler
}

func (ch *RawTCPServerHandler) Serve(ctx context.Context, conn *Conn) {
	if h := ch.Handler; h != nil {
		h.ServeTCP(ctx, conn)
	}
}

func NewServerHandler(h RawTCPHandler) ServerHandler {
	return &RawTCPServerHandler{
		Handler: h,
	}
}
