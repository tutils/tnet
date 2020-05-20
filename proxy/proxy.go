package proxy

import (
	"bufio"
	"context"
	"github.com/tutils/tnet/crypt/xor"
	"github.com/tutils/tnet/tcp"
	"io"
	"net"
	"sync"
	"testing"
)

type connIdKey struct{}

var connId int64 = 0

func connIdGen() int64 {
	connId++
	return connId
}

type proxyHandler struct {
	t       *testing.T
	s       *tcp.Server
	tunw    io.Writer
	connMap *sync.Map
}

func (h *proxyHandler) ServeTCP(ctx context.Context, conn *tcp.Conn) {
	// new proxy connection
	connr := conn.Reader()
	tunw := bufio.NewWriterSize(h.tunw, 5<<10) // TODO: use pool

	// conn_reader -> tun_writer
	connId := ctx.Value(connIdKey{}).(int64)
	h.connMap.Store(connId, conn)

	if err := packHeader(tunw, CmdConnect); err != nil {
		return
	}
	if err := packBodyConnect(tunw, connId); err != nil {
		return
	}
	if err := tunw.Flush(); err != nil {
		return
	}

	buf := make([]byte, 4<<10)
	for {
		n, err := connr.Read(buf)
		if err != nil {
			return
		}
		select {
		case <-ctx.Done():
			return // remote peer close
		default:
		}
		if n <= 0 {
			continue
		}
		if err := packHeader(tunw, CmdSend); err != nil {
			return
		}
		if err := packBodySend(tunw, connId, buf[:n]); err != nil {
			return
		}
		if err := tunw.Flush(); err != nil {
			return
		}
	}
}

type tunHandler struct {
	t       *testing.T
	connMap sync.Map
}

var cr = xor.NewCrypt(812734)

func (h *tunHandler) ServeTun(r io.Reader, w io.Writer) {
	// tcp tunnel has been setup
	tcph := &proxyHandler{
		t:       h.t,
		tunw:    cr.NewEncoder(w),
		connMap: &h.connMap,
	}

	s := tcp.NewServer(
		tcp.WithListenAddress(":56081"),
		tcp.WithServerHandler(tcp.NewServerHandler(tcph)),
		tcp.WithConnContextFunc(func(ctx context.Context, c net.Conn) context.Context {
			return context.WithValue(ctx, connIdKey{}, connIdGen())
		}),
	)
	tcph.s = s
	go s.ListenAndServe()
	defer s.Shutdown(context.Background())

	// tun_reader -> conn_writer
	tunr := cr.NewDecoder(r)
	for {
		cmd, err := unpackHeader(tunr)
		if err != nil {
			return
		}
		switch cmd {
		case CmdSend:
			f := func(connId int64) io.Writer {
				v, ok := h.connMap.Load(connId)
				if !ok {
					return nil
				}
				return v.(*tcp.Conn).Writer()
			}
			_, err := unpackBodySend(tunr, f)
			if err != nil {
				return
			}
		case CmdClose:
			connId, err := unpackBodyClose(tunr)
			if err != nil {
				return
			}
			v, ok := h.connMap.Load(connId)
			if !ok {
				break // ignore
			}
			conn := v.(*tcp.Conn)
			h.connMap.Delete(connId)
			conn.CancelContext()
			conn.AbortPendingRead()
		}
	}
}

type Proxy struct {
	opts   ProxyOptions
	tcpSrv *tcp.Server
}

func (p *Proxy) DialTunnel() error {
	return nil
}

func NewProxy(opts ...ProxyOption) *Proxy {
	opt := newProxyOptions(opts...)

	return &Proxy{
		opts: *opt,
	}
}
