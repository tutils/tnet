package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/tutils/tnet"
	"github.com/tutils/tnet/endpoint/common"
	"github.com/tutils/tnet/tcp"
)

type tcpConnDataKey struct{}

type tcpConnData struct {
	tunID   int64
	connID  int64
	writeCh chan []byte
	closeCh chan struct{}
}

// tcpHandler
type tcpHandler struct {
	tunw    io.Writer // SyncWriter
	connMap *sync.Map
}

func (e *tcpHandler) ServeTCP(ctx context.Context, conn tcp.Conn) {
	connr := conn.Reader()
	tunw := e.tunw
	connMap := e.connMap

	connData := ctx.Value(tcpConnDataKey{}).(*tcpConnData)
	tunID := connData.tunID
	connID := connData.connID
	connMap.Store(connID, connData)
	log.Printf("new agent connection, connID %d:%d", tunID, connID)
	defer log.Printf("agent connection closed, connID %d:%d", tunID, connID)

	tunwbuf := &bytes.Buffer{} // TODO: use pool
	if err := common.PackHeader(tunwbuf, common.CmdConnectResult); err != nil {
		log.Println("packHeader err", err)
		return
	}
	if err := common.PackBodyConnectResult(tunwbuf, connID, nil); err != nil {
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
			if err := common.PackHeader(tunwbuf, common.CmdClose); err != nil {
				log.Println("packHeader err", err)
				break
			}
			if err := common.PackBodyClose(tunwbuf, connID); err != nil {
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
		if err := common.PackHeader(tunwbuf, common.CmdSend); err != nil {
			log.Println("packHeader err", err)
			return
		}
		if err := common.PackBodySend(tunwbuf, connID, buf[:n]); err != nil {
			log.Println("packBodySend err", err)
			return
		}
		if _, err := tunw.Write(tunwbuf.Bytes()); err != nil {
			log.Println("write tun err", err)
			return
		}
	}
}

// Agent for connecting remote tcp server
type Agent struct {
	opts Options
}

// New create a new Endpoint
func New(opts ...Option) *Agent {
	opt := newOptions(opts...)
	return &Agent{
		opts: *opt,
	}
}

// Serve starts agent
func (a *Agent) Serve() error {
	if tunClient := a.opts.tunClient; tunClient != nil {
		log.Println("start tun client (reverse mode)")
		defer log.Println("tun client exit")
		return tunClient.DialAndServe(a)
	}

	if tunServer := a.opts.tunServer; tunServer != nil {
		log.Println("start tun server")
		defer log.Println("tun server exit")
		return tunServer.ListenAndServe(a)
	}

	return fmt.Errorf("neither tunnel client nor server is configured")
}

// ServeTun handles the tune
func (a *Agent) ServeTun(ctx context.Context, r io.Reader, w io.Writer) {
	log.Println("new tun connection")
	defer log.Println("tun connection closed")

	// new tun connection
	var tunr io.Reader
	if crypt := a.opts.tunCrypt; crypt != nil {
		tunr = crypt.NewDecoder(r)
	} else {
		tunr = r
	}

	// recv config: connect to
	cmd, err := common.UnpackHeader(tunr)
	if err != nil {
		log.Println("unpackHeader err", err)
		return
	}
	if cmd != common.CmdConfig {
		log.Println("cmd err")
		return
	}
	connectAddr, err := common.UnpackBodyConfig(tunr)
	if err != nil {
		log.Println("unpackBodyConfig err", err)
		return
	}
	log.Printf("Read CmdConfig, connectAddr %s", connectAddr)

	var tunw io.Writer
	if crypt := a.opts.tunCrypt; crypt != nil {
		tunw = crypt.NewEncoder(w)
	} else {
		tunw = w
	}
	tunw = tnet.NewSyncWriter(tunw)

	// sync tunID
	isServer := a.opts.tunServer != nil
	tunID, err := common.SyncTunID(ctx, isServer, tunr, tunw)
	if err != nil {
		return
	}

	var connMap sync.Map

	tcph := &tcpHandler{
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
		cmd, err := common.UnpackHeader(tunr)
		if err != nil {
			log.Println("unpackHeader err", err)
			return
		}
		switch cmd {
		case common.CmdConnect:
			connID, err := common.UnpackBodyConnect(tunr)
			log.Printf("Read CmdConnect, connID %d:%d", tunID, connID)
			if err != nil {
				log.Println("unpackBodyConnect err", err)
				return
			}
			ctx := context.Background()
			data := &tcpConnData{
				tunID:   tunID,
				connID:  connID,
				writeCh: make(chan []byte, 1<<8),
				closeCh: make(chan struct{}),
			}
			ctx = context.WithValue(ctx, tcpConnDataKey{}, data)
			go func() {
				buf := &bytes.Buffer{} // TODO: use pool
				if err := c.DialAndServe(ctx); err != nil {
					buf.Reset()
					if err := common.PackHeader(buf, common.CmdConnectResult); err != nil {
						log.Println("packHeader err", err)
						return
					}
					if err := common.PackBodyConnectResult(buf, connID, err); err != nil {
						log.Println("packBodyConnectResult err", err)
						return
					}
					if _, err := tunw.Write(buf.Bytes()); err != nil {
						log.Println("write tun err", err)
						return
					}
				}
			}()
		case common.CmdSend:
			connID, data, err := common.UnpackBodySend(tunr)
			log.Printf("Read CmdSend, connID %d:%d, %d bytes", tunID, connID, len(data))
			if err != nil {
				log.Println("unpackBodySend err", err)
				return
			}
			v, ok := connMap.Load(connID)
			if !ok {
				log.Printf("connID %d:%d not found", tunID, connID)
				break // ignore
			}
			v.(*tcpConnData).writeCh <- data
		case common.CmdClose:
			connID, err := common.UnpackBodyClose(tunr)
			log.Printf("Read CmdClose, connID %d:%d,", tunID, connID)
			if err != nil {
				log.Println("unpackBodyClose err", err)
				return
			}
			v, ok := connMap.Load(connID)
			if !ok {
				log.Printf("connID %d:%d not found", tunID, connID)
				break // ignore
			}
			close(v.(*tcpConnData).closeCh)
		}
	}
}
