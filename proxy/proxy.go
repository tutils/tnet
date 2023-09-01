package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/tutils/tnet"
	"github.com/tutils/tnet/counter"
	"github.com/tutils/tnet/crypt"
	"github.com/tutils/tnet/tcp"
	"github.com/tutils/tnet/tun"
)

type counterWriter struct {
	w io.Writer
	c counter.Counter
}

func (w *counterWriter) Write(p []byte) (n int, err error) {
	n, err = w.w.Write(p)
	if err == nil && n >= 0 {
		w.c.Add(int64(n))
	}
	return n, err
}

type counterReader struct {
	r io.Reader
	c counter.Counter
}

func (r *counterReader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	if err == nil && n >= 0 {
		r.c.Add(int64(n))
	}
	return n, err
}

var suffix = []string{"B", "KB", "MB", "GB", "TB"}

func humanReadable(bytes uint64) string {
	return fmt.Sprintf("%d B", bytes)
}

type proxyConnDataKey struct{}

type proxyConnData struct {
	connID       int64
	connectResCh chan error
	writeCh      chan []byte
	closeCh      chan struct{}
}

// proxyHandler
type proxyHandler struct {
	tunw    io.Writer // SyncWriter
	tunID   int64
	connMap *sync.Map
}

// ServeTCP called from multiple goroutines
func (h *proxyHandler) ServeTCP(ctx context.Context, conn tcp.Conn) {
	// new proxy connection
	connr := conn.Reader()
	tunw := h.tunw
	connMap := h.connMap

	// conn_reader -> tun_writer
	connData := ctx.Value(proxyConnDataKey{}).(*proxyConnData)
	connID := connData.connID
	connMap.Store(connID, connData)
	log.Printf("new proxy connection, connID %d:%d", h.tunID, connID)
	defer log.Printf("proxy connection closed, connID %d:%d", h.tunID, connID)

	tunwbuf := &bytes.Buffer{} // TODO: use pool
	if err := packHeader(tunwbuf, CmdConnect); err != nil {
		log.Println("packHeader err", err)
		return
	}
	if err := packBodyConnect(tunwbuf, connID); err != nil {
		log.Println("packBodyConnect err", err)
		return
	}
	log.Printf("Write CmdConnect, connID %d:%d", h.tunID, connID)
	if _, err := tunw.Write(tunwbuf.Bytes()); err != nil {
		log.Println("write tun err", err)
		return
	}

	if err := <-connData.connectResCh; err != nil {
		return
	}

	done := make(chan struct{})
	defer func() {
		connMap.Delete(connID)
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
			tunwbuf.Reset()
			if err := packHeader(tunwbuf, CmdClose); err != nil {
				log.Println("packHeader err", err)
				break
			}
			if err := packBodyClose(tunwbuf, connID); err != nil {
				log.Println("packBodyClose err", err)
				break
			}
			log.Printf("Write CmdClose, connID %d:%d", h.tunID, connID)
			if _, err := tunw.Write(tunwbuf.Bytes()); err != nil {
				log.Println("write tun err", err)
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
					log.Println("write conn err", err)
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

	buf := make([]byte, 40<<10)
	for {
		n, err := connr.Read(buf)
		if err != nil {
			log.Println("read conn err", err)
			return
		}

		select {
		case <-connData.closeCh:
			return // remote peer close
		default:
		}

		tunwbuf.Reset()
		if err := packHeader(tunwbuf, CmdSend); err != nil {
			log.Println("packHeader err", err)
			return
		}
		if err := packBodySend(tunwbuf, connID, buf[:n]); err != nil {
			log.Println("packBodySend err", err)
			return
		}
		if cw, ok := tunw.(*counterWriter); ok {
			log.Printf("Write CmdSend, connID %d:%d, %d bytes, download %s/s", h.tunID, connID, tunwbuf.Len(), humanReadable(uint64(cw.c.IncreaceRatePerSec())))
		} else {
			log.Printf("Write CmdSend, connID %d:%d, %d bytes", h.tunID, connID, tunwbuf.Len())
		}
		if _, err := tunw.Write(tunwbuf.Bytes()); err != nil {
			log.Println("write tun err", err)
			return
		}
	}
}

// tunClientHandler
type tunClientHandler struct {
	crypt           crypt.Crypt
	proxyAddr       string
	connectAddr     string
	downloadCounter counter.Counter
	uploadCounter   counter.Counter
}

func (h *tunClientHandler) ServeTun(ctx context.Context, r io.Reader, w io.Writer) {
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
	if h.uploadCounter != nil {
		tunw = &counterWriter{w: tunw, c: h.uploadCounter}
	}

	var tunr io.Reader
	if h.crypt != nil {
		tunr = h.crypt.NewDecoder(r)
	} else {
		tunr = r
	}
	if h.downloadCounter != nil {
		tunr = &counterReader{r: tunr, c: h.downloadCounter}
	}

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
	log.Printf("Write CmdConfig, connectAddr %s", h.connectAddr)
	if _, err := tunw.Write(buf.Bytes()); err != nil {
		log.Println("write tun err", err)
		return
	}

	// recv tunID
	cmd, err := unpackHeader(tunr)
	if err != nil {
		log.Println("unpackHeader err", err)
		return
	}
	if cmd != CmdTunID {
		log.Println("cmd err")
		return
	}
	tunID, err := unpackBodyTunID(tunr)
	if err != nil {
		log.Println("unpackBodyTunID err", err)
		return
	}

	tcph := &proxyHandler{
		tunw:    tunw,
		tunID:   tunID,
		connMap: &connMap,
	}

	var connID int64

	s := tcp.NewServer(
		tcp.WithListenAddress(h.proxyAddr),
		tcp.WithServerHandler(tcp.NewRawTCPConnHandler(tcph)),
		tcp.WithServerConnContextFunc(func(ctx context.Context, c net.Conn) context.Context {
			connID++
			data := &proxyConnData{
				connID:       connID,
				connectResCh: make(chan error, 1),
				writeCh:      make(chan []byte, 1<<8),
				closeCh:      make(chan struct{}),
			}
			return context.WithValue(ctx, proxyConnDataKey{}, data)
		}),
		tcp.WithServerKeepAlivePeriod(time.Second*15),
		tcp.WithServerKeepAliveCount(3),
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.ListenAndServe()
	}()
	defer s.Shutdown(context.Background())

	// tun_reader -> conn_writer
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
			connID, connectResult, err := unpackBodyConnectResult(tunr)
			log.Printf("Read CmdConnectResult, connID %d:%d, %v", tunID, connID, connectResult)
			if err != nil {
				log.Println("unpackBodyConnectResult err", err)
				return
			}
			v, ok := connMap.Load(connID)
			if !ok {
				log.Printf("connID %d:%d not found", tunID, connID)
				break // ignore
			}
			v.(*proxyConnData).connectResCh <- connectResult

		case CmdSend:
			connID, data, err := unpackBodySend(tunr)
			if cr, ok := tunr.(*counterReader); ok {
				log.Printf("Read CmdSend, connID %d:%d, %d bytes, upload %s/s", tunID, connID, len(data), humanReadable(uint64(cr.c.IncreaceRatePerSec())))
			} else {
				log.Printf("Read CmdSend, connID %d:%d, %d bytes", tunID, connID, len(data))
			}
			if err != nil {
				log.Println("unpackBodySend err", err)
				return
			}
			v, ok := connMap.Load(connID)
			if !ok {
				log.Printf("connID %d:%d not found", tunID, connID)
				break // ignore
			}
			v.(*proxyConnData).writeCh <- data

		case CmdClose:
			connID, err := unpackBodyClose(tunr)
			log.Printf("Read CmdClose, connID %d:%d", tunID, connID)
			if err != nil {
				log.Println("unpackBodyClose err", err)
				return
			}
			v, ok := connMap.Load(connID)
			if !ok {
				log.Printf("connID %d:%d not found", tunID, connID)
				break // ignore
			}
			close(v.(*proxyConnData).closeCh)
		}
	}
}

// NewTunClientHandler create a new tunnel handler for proxy
func NewTunClientHandler() tun.Handler {
	return &tunClientHandler{}
}

// Proxy for proxying remote tcp server to local address
type Proxy struct {
	opts ProxyOptions
}

// DialAndServe starts proxy
func (p *Proxy) DialAndServe() error {
	log.Println("start tun client")
	defer log.Println("tun client exit")
	return p.opts.tun.DialAndServe()
}

// NewProxy create a new proxy
func NewProxy(opts ...ProxyOption) *Proxy {
	opt := newProxyOptions(opts...)

	p := &Proxy{
		opts: *opt,
	}

	if h, ok := p.opts.tun.Handler().(*tunClientHandler); ok {
		h.crypt = opt.tunCrypt
		h.proxyAddr = opt.listenAddr
		h.connectAddr = opt.connectAddr
		h.downloadCounter = opt.downloadCounter
		h.uploadCounter = opt.uploadCounter
	}

	return p
}
