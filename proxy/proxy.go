package proxy

import (
	"bytes"
	"context"
	"github.com/tutils/tnet"
	"github.com/tutils/tnet/crypt"
	"github.com/tutils/tnet/tcp"
	"github.com/tutils/tnet/tun"
	"io"
	"log"
	"net"
	"sync"
)

var connId_ int64 = 0

func connIdGen() int64 {
	connId_++
	return connId_
}

type proxyConnDataKey struct{}

type proxyConnData struct {
	connId       int64
	connectResCh chan error
	writeCh      chan []byte
	closeCh      chan struct{}
}

// proxyHandler
type proxyHandler struct {
	tunw    io.Writer // SyncWriter
	connMap *sync.Map
}

// called from multiple goroutines
func (h *proxyHandler) ServeTCP(ctx context.Context, conn tcp.Conn) {
	log.Println("new proxy connection")
	defer log.Println("proxy connection closed")

	// new proxy connection
	connr := conn.Reader()
	tunw := h.tunw
	connMap := h.connMap

	// conn_reader -> tun_writer
	connData := ctx.Value(proxyConnDataKey{}).(*proxyConnData)
	connId := connData.connId
	connMap.Store(connId, connData)

	buftunw := &bytes.Buffer{} // TODO: use pool
	if err := packHeader(buftunw, CmdConnect); err != nil {
		return
	}
	if err := packBodyConnect(buftunw, connId); err != nil {
		return
	}
	if _, err := tunw.Write(buftunw.Bytes()); err != nil {
		return
	}

	if err := <-connData.connectResCh; err != nil {
		return
	}

	done := make(chan struct{})
	defer func() {
		connMap.Delete(connId)
	LOOP:
		for {
			// clean writeCh
			select {
			case <-connData.writeCh:
			default:
				break LOOP
			}
		}

		close(done)
		select {
		case <-connData.closeCh:
		default:
			buftunw.Reset()
			if err := packHeader(buftunw, CmdClose); err != nil {
				break
			}
			if err := packBodyClose(buftunw, connId); err != nil {
				break
			}
			if _, err := tunw.Write(buftunw.Bytes()); err != nil {
				break
			}
		}
	}()

	go func() {
		connw := conn.Writer()
		for {
			select {
			case data, ok := <-connData.writeCh:
				if !ok {
					return
				}
				if _, err := connw.Write(data); err != nil {
					return
				}
			case <-connData.closeCh:
				conn.CancelContext()
				conn.AbortPendingRead()
				return
			case <-done:
				return
			}
		}
	}()

	buf := make([]byte, 4<<10)
	for {
		n, err := connr.Read(buf)
		if err != nil {
			return
		}

		select {
		case <-connData.closeCh:
			return // remote peer close
		default:
		}

		buftunw.Reset()
		if err := packHeader(buftunw, CmdSend); err != nil {
			return
		}
		if err := packBodySend(buftunw, connId, buf[:n]); err != nil {
			return
		}
		if _, err := tunw.Write(buftunw.Bytes()); err != nil {
			return
		}
	}
}

// tunClientHandler
type tunClientHandler struct {
	crypt       crypt.Crypt
	proxyAddr   string
	connectAddr string
}

func (h *tunClientHandler) ServeTun(r io.Reader, w io.Writer) {
	log.Println("new tun connection")
	defer log.Println("tun connection closed")

	// tcp tunnel has been setup
	var tunw io.Writer
	if h.crypt != nil {
		tunw = h.crypt.NewEncoder(w)
	} else {
		tunw = w
	}
	tunw = tnet.NewSyncWriter(tunw)

	var connMap sync.Map

	// send config
	buf := &bytes.Buffer{} // TODO: use pool
	if err := packHeader(buf, CmdConfig); err != nil {
		log.Println("packHeader err", err)
		return
	}
	if err := packBodyConfig(buf, h.connectAddr); err != nil {
		log.Println("packBodyConfig err", err)
		return
	}
	if _, err := tunw.Write(buf.Bytes()); err != nil {
		log.Println("write tun err", err)
		return
	}

	tcph := &proxyHandler{
		tunw:    tunw,
		connMap: &connMap,
	}

	s := tcp.NewServer(
		tcp.WithListenAddress(h.proxyAddr),
		tcp.WithServerHandler(tcp.NewRawTCPHandler(tcph)),
		tcp.WithServerConnContextFunc(func(ctx context.Context, c net.Conn) context.Context {
			data := &proxyConnData{
				connId:       connIdGen(),
				connectResCh: make(chan error, 1),
				writeCh:      make(chan []byte, 1<<8),
				closeCh:      make(chan struct{}),
			}
			return context.WithValue(ctx, proxyConnDataKey{}, data)
		}),
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.ListenAndServe()
	}()
	defer s.Shutdown(context.Background())

	// tun_reader -> conn_writer
	var tunr io.Reader
	if h.crypt != nil {
		tunr = h.crypt.NewDecoder(r)
	} else {
		tunr = r
	}

	for {
		select {
		case <-errCh:
			return
		default:
		}

		cmd, err := unpackHeader(tunr)
		if err != nil {
			log.Println("unpackHeader err", err)
			return
		}
		switch cmd {
		case CmdConnectResult:
			connId, connectResult, err := unpackBodyConnectResult(tunr)
			if err != nil {
				log.Println("unpackBodyConnectResult err", err)
				return
			}
			v, ok := connMap.Load(connId)
			if !ok {
				log.Println("connId not found")
				break // ignore
			}
			v.(*proxyConnData).connectResCh <- connectResult

		case CmdSend:
			connId, data, err := unpackBodySend(tunr)
			if err != nil {
				log.Println("unpackBodySend err", err)
				return
			}
			v, ok := connMap.Load(connId)
			if !ok {
				log.Println("connId not found")
				break // ignore
			}
			v.(*proxyConnData).writeCh <- data

		case CmdClose:
			connId, err := unpackBodyClose(tunr)
			if err != nil {
				log.Println("unpackBodyClose err", err)
				return
			}
			v, ok := connMap.Load(connId)
			if !ok {
				log.Println("connId not found")
				break // ignore
			}
			close(v.(*proxyConnData).closeCh)
		}
	}
}

func NewTunClientHandler() tun.Handler {
	return &tunClientHandler{}
}

// Proxy
type Proxy struct {
	opts ProxyOptions
}

func (p *Proxy) DialAndServe() error {
	log.Println("start tun client")
	defer log.Println("tun client exit")
	return p.opts.tun.DialAndServe()
}

func NewProxy(opts ...ProxyOption) *Proxy {
	opt := newProxyOptions(opts...)

	p := &Proxy{
		opts: *opt,
	}

	if h, ok := p.opts.tun.Handler().(*tunClientHandler); ok {
		h.crypt = opt.tunCrypt
		h.proxyAddr = opt.listenAddr
		h.connectAddr = opt.connectAddr
	}

	return p
}
