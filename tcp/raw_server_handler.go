package tcp

import "context"

// RawTCPHandler is raw tcp handler
type RawTCPHandler interface {
	ServeTCP(ctx context.Context, conn Conn)
}

// RawTCPConnHandler is raw tcp connection handler
type RawTCPConnHandler struct {
	Handler RawTCPHandler
}

// ServeConn serves new connection
func (ch *RawTCPConnHandler) ServeConn(ctx context.Context, conn Conn) {
	if h := ch.Handler; h != nil {
		h.ServeTCP(ctx, conn)
	}
}

// NewRawTCPConnHandler creates a new raw tcp connection handler
func NewRawTCPConnHandler(h RawTCPHandler) ConnHandler {
	return &RawTCPConnHandler{
		Handler: h,
	}
}
