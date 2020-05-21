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
)

type endpointConnDataKey struct{}

type endpointConnData struct {
	connId  int64
	writeCh chan []byte
	closeCh chan struct{}
}

// endpointHandler
type endpointHandler struct {
	tunw    io.Writer // SyncWriter
	connMap *sync.Map
}

func (e *endpointHandler) ServeConn(ctx context.Context, conn tcp.Conn) {
	log.Println("new endpoint connection")
	defer log.Println("endpoint connection closed")

	connr := conn.Reader()
	tunw := e.tunw
	connMap := e.connMap

	connData := ctx.Value(endpointConnDataKey{}).(*endpointConnData)
	connId := connData.connId
	connMap.Store(connId, connData)

	buftunw := &bytes.Buffer{} // TODO: use pool
	if err := packHeader(buftunw, CmdConnectResult); err != nil {
		log.Println("packHeader err", err)
		return
	}
	if err := packBodyConnectResult(buftunw, connId, nil); err != nil {
		log.Println("packBodyConnectResult err", err)
		return
	}
	if _, err := tunw.Write(buftunw.Bytes()); err != nil {
		log.Println("write tun err", err)
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
				log.Println("packHeader err", err)
				break
			}
			if err := packBodyClose(buftunw, connId); err != nil {
				log.Println("packBodyClose err", err)
				break
			}
			if _, err := tunw.Write(buftunw.Bytes()); err != nil {
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

	buf := make([]byte, 4<<10)
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

		buftunw.Reset()
		if err := packHeader(buftunw, CmdSend); err != nil {
			log.Println("packHeader err", err)
			return
		}
		if err := packBodySend(buftunw, connId, buf[:n]); err != nil {
			log.Println("packBodySend err", err)
			return
		}
		if _, err := tunw.Write(buftunw.Bytes()); err != nil {
			log.Println("write tun err", err)
			return
		}
	}
}

// tunServerHandler
type tunServerHandler struct {
	crypt crypt.Crypt
}

func (h *tunServerHandler) ServeTun(r io.Reader, w io.Writer) {
	log.Println("new tun connection")
	defer log.Println("tun connection closed")

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

	var connMap sync.Map

	tcph := &endpointHandler{
		tunw:    tunw,
		connMap: &connMap,
	}

	c := tcp.NewClient(
		tcp.WithConnectAddress(connectAddr),
		tcp.WithClientHandler(tcph),
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
			connId, err := unpackBodyConnect(tunr)
			if err != nil {
				log.Println("unpackBodyConnect err", err)
				return
			}
			ctx := context.Background()
			data := &endpointConnData{
				connId:  connId,
				writeCh: make(chan []byte, 1<<8),
				closeCh: make(chan struct{}),
			}
			ctx = context.WithValue(ctx, endpointConnDataKey{}, data)
			go func() {
				if err := c.DialAndServe(ctx); err != nil {
					buftunw := &bytes.Buffer{} // TODO: use pool
					if err := packHeader(buftunw, CmdConnectResult); err != nil {
						log.Println("packHeader err", err)
						return
					}
					if err := packBodyConnectResult(buftunw, connId, err); err != nil {
						log.Println("packBodyConnectResult err", err)
						return
					}
					if _, err := tunw.Write(buftunw.Bytes()); err != nil {
						log.Println("write tun err", err)
						return
					}
				}
			}()
		case CmdSend:
			connId, data, err := unpackBodySend(tunr)
			if err != nil {
				log.Println("unpackBodySend err", err)
				return
			}
			v, ok := connMap.Load(connId)
			if !ok {
				log.Println("connId not found")
				break  // ignore
			}
			v.(*endpointConnData).writeCh <- data
		case CmdClose:
			connId, err := unpackBodyClose(tunr)
			if err != nil {
				log.Println("unpackBodyClose err", err)
				return
			}
			v, ok := connMap.Load(connId)
			if !ok {
				log.Println("connId not found")
				break  // ignore
			}
			close(v.(*endpointConnData).closeCh)
		}
	}
}

func NewTunServerHandler() tun.Handler {
	return &tunServerHandler{}
}

// Endpoint
type Endpoint struct {
	opts EndpointOptions
}

func (p *Endpoint) ListenAndServe() error {
	log.Println("start tun server")
	defer log.Println("tun server exit")
	return p.opts.tun.ListenAndServe()
}

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
