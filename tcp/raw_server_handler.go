package tcp

import "context"

// raw tcp handler
type RawTCPHandler interface {
	ServeTCP(ctx context.Context, conn Conn)
}

// raw tcp connection handler
type RawTCPConnHandler struct {
	Handler RawTCPHandler
}

func (ch *RawTCPConnHandler) ServeConn(ctx context.Context, conn Conn) {
	if h := ch.Handler; h != nil {
		h.ServeTCP(ctx, conn)
	}
}

// create a raw tcp connection handler
func NewRawTCPConnHandler(h RawTCPHandler) ConnHandler {
	return &RawTCPConnHandler{
		Handler: h,
	}
}
