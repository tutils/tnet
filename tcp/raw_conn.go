package tcp

import "context"

type RawTCPHandler interface {
	ServeTCP(ctx context.Context, conn *conn)
}

type RawTCPConnectionHandler struct {
	Handler RawTCPHandler
}

func (ch *RawTCPConnectionHandler) Serve(ctx context.Context, conn *conn) {
	if h := ch.Handler; h != nil {
		h.ServeTCP(ctx, conn)
	}
}

func NewRawTCPConnectionHandler(h RawTCPHandler) ConnectionHandler {
	return &RawTCPConnectionHandler{
		Handler: h,
	}
}
