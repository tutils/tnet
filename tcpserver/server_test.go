package tcpserver

import (
	"context"
	"testing"
	"time"
)

type handler struct {
	t *testing.T
}

func (h *handler) ServeTCP(ctx context.Context, conn *conn) {
	conn.r.setReadLimit(maxInt64)

	for {
		ln, _, err := conn.bufr.ReadLine()
		if err != nil {
			return
		} else {
			conn.bufw.WriteString("OK\n")
			conn.bufw.Flush()
			h.t.Logf("%s", string(ln))
		}
	}
}

func TestListenAndServe(t *testing.T) {
	h := &handler{t: t}
	ch := &RawTCPConnectionHandler{Handler: h}
	server := &Server{Addr: ":8080", Handler: ch}
	go server.ListenAndServe()
	time.Sleep(time.Second * 5)
	t.Logf("shutdown ... %#v", server.activeConn)
	server.Shutdown(context.Background())
	t.Logf("done")
	time.Sleep(time.Second * 5)
	t.Logf("%#v", server.activeConn)
}
