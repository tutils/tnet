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
	"sync"
	"time"
)

type endpointConnDataKey struct{}

type endpointConnData struct {
	tunID   int64
	connID  int64
	writeCh chan []byte
	closeCh chan struct{}
}

// endpointHandler
type endpointHandler struct {
	tunw    io.Writer // SyncWriter
	connMap *sync.Map
}

func (e *endpointHandler) ServeTCP(ctx context.Context, conn tcp.Conn) {
	connr := conn.Reader()
	tunw := e.tunw
	connMap := e.connMap

	connData := ctx.Value(endpointConnDataKey{}).(*endpointConnData)
	tunID := connData.tunID
	connID := connData.connID
	connMap.Store(connID, connData)
	log.Printf("new endpoint connection, connID %d:%d", tunID, connID)
	defer log.Printf("endpoint connection closed, connID %d:%d", tunID, connID)

	tunwbuf := &bytes.Buffer{} // TODO: use pool
	if err := packHeader(tunwbuf, CmdConnectResult); err != nil {
		log.Println("packHeader err", err)
		return
	}
	if err := packBodyConnectResult(tunwbuf, connID, nil); err != nil {
		log.Println("packBodyConnectResult err", err)
		return
	}
	if _, err := tunw.Write(tunwbuf.Bytes()); err != nil {
		log.Println("write tun err", err)
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
		if _, err := tunw.Write(tunwbuf.Bytes()); err != nil {
			log.Println("write tun err", err)
			return
		}
	}
}

// tunServerHandler
type tunServerHandler struct {
	crypt crypt.Crypt
}

func (h *tunServerHandler) ServeTun(ctx context.Context, r io.Reader, w io.Writer) {
	log.Println("new tun connection")
	defer log.Println("tun connection closed")

	tunID := ctx.Value(tun.ConnIDContextKey{}).(int64)

	// new tun connection
	var tunr io.Reader
	if h.crypt != nil {
		tunr = h.crypt.NewDecoder(r)
	} else {
		tunr = r
	}

	// recv config
	cmd, err := unpackHeader(tunr)
	if err != nil {
		log.Println("unpackHeader err", err)
		return
	}
	if cmd != CmdConfig {
		log.Println("cmd err")
		return
	}
	connectAddr, err := unpackBodyConfig(tunr)
	if err != nil {
		log.Println("unpackBodyConfig err", err)
		return
	}

	var tunw io.Writer
	if h.crypt != nil {
		tunw = h.crypt.NewEncoder(w)
	} else {
		tunw = w
	}
	tunw = tnet.NewSyncWriter(tunw)

	// send tunConnID
	tunwbuf := &bytes.Buffer{} // TODO: use pool
	if err := packHeader(tunwbuf, CmdTunID); err != nil {
		log.Println("packHeader err", err)
		return
	}
	if err := packBodyTunID(tunwbuf, tunID); err != nil {
		log.Println("packBodyTunID err", err)
		return
	}
	if _, err := tunw.Write(tunwbuf.Bytes()); err != nil {
		log.Println("write tun err", err)
		return
	}

	var connMap sync.Map

	tcph := &endpointHandler{
		tunw:    tunw,
		connMap: &connMap,
	}

	c := tcp.NewClient(
		tcp.WithConnectAddress(connectAddr),
		tcp.WithClientHandler(tcp.NewRawTCPConnHandler(tcph)),
		tcp.WithClientKeepAlivePeriod(time.Second*15),
		tcp.WithClientKeepAliveCount(3),
	)
	defer c.Shutdown(context.Background())

	for {
		cmd, err := unpackHeader(tunr)
		if err != nil {
			log.Println("unpackHeader err", err)
			return
		}
		switch cmd {
		case CmdConnect:
			connID, err := unpackBodyConnect(tunr)
			log.Printf("CmdConnect, connID %d:%d", tunID, connID)
			if err != nil {
				log.Println("unpackBodyConnect err", err)
				return
			}
			ctx := context.Background()
			data := &endpointConnData{
				tunID:   tunID,
				connID:  connID,
				writeCh: make(chan []byte, 1<<8),
				closeCh: make(chan struct{}),
			}
			ctx = context.WithValue(ctx, endpointConnDataKey{}, data)
			go func() {
				if err := c.DialAndServe(ctx); err != nil {
					tunwbuf.Reset()
					if err := packHeader(tunwbuf, CmdConnectResult); err != nil {
						log.Println("packHeader err", err)
						return
					}
					if err := packBodyConnectResult(tunwbuf, connID, err); err != nil {
						log.Println("packBodyConnectResult err", err)
						return
					}
					if _, err := tunw.Write(tunwbuf.Bytes()); err != nil {
						log.Println("write tun err", err)
						return
					}
				}
			}()
		case CmdSend:
			connID, data, err := unpackBodySend(tunr)
			log.Printf("CmdSend, connID %d:%d, %d bytes", tunID, connID, len(data))
			if err != nil {
				log.Println("unpackBodySend err", err)
				return
			}
			v, ok := connMap.Load(connID)
			if !ok {
				log.Printf("connID %d:%d not found", tunID, connID)
				break // ignore
			}
			v.(*endpointConnData).writeCh <- data
		case CmdClose:
			connID, err := unpackBodyClose(tunr)
			log.Printf("CmdClose, connID %d:%d,", tunID, connID)
			if err != nil {
				log.Println("unpackBodyClose err", err)
				return
			}
			v, ok := connMap.Load(connID)
			if !ok {
				log.Printf("connID %d:%d not found", tunID, connID)
				break // ignore
			}
			close(v.(*endpointConnData).closeCh)
		}
	}
}

// NewTunServerHandler create a new tunnel handler for endpoint
func NewTunServerHandler() tun.Handler {
	return &tunServerHandler{}
}

// Endpoint for connecting remote tcp server
type Endpoint struct {
	opts EndpointOptions
}

// ListenAndServe starts endpoint
func (p *Endpoint) ListenAndServe() error {
	log.Println("start tun server")
	defer log.Println("tun server exit")
	return p.opts.tun.ListenAndServe()
}

// NewEndpoint create a new Endpoint
func NewEndpoint(opts ...EndpointOption) *Endpoint {
	opt := newEndpointOptions(opts...)

	p := &Endpoint{
		opts: *opt,
	}

	if h, ok := p.opts.tun.Handler().(*tunServerHandler); ok {
		h.crypt = opt.tunCrypt
	}

	return p
}
