package agent

import (
	"bytes"
	"context"
	"io"
	"log"
	"sync"
	"time"

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
			// check if connection is closed
			select {
			case <-connData.closeCh:
				log.Printf("read conn abort: proxy connection closed, connID %d:%d", tunID, connID)
			default:
				log.Printf("read conn err: %v, connID %d:%d", err, tunID, connID)
			}
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

func (h *agentTunHandler) agentTCP(ctx context.Context, tunID int64, tunr io.Reader, tunw io.Writer) {
	connectAddr, err := common.UnpackBodyConfig(tunr)
	if err != nil {
		log.Println("unpackBodyConfig err", err)
		return
	}
	log.Printf("Read CmdConfig, connectAddr %s", connectAddr)

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
			if err != nil {
				log.Println("unpackBodyConnect err", err)
				return
			}
			log.Printf("Read CmdConnect, connID %d:%d", tunID, connID)
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
			if err != nil {
				log.Println("unpackBodySend err", err)
				return
			}
			log.Printf("Read CmdSend, connID %d:%d, %d bytes", tunID, connID, len(data))
			v, ok := connMap.Load(connID)
			if !ok {
				log.Printf("connID %d:%d not found", tunID, connID)
				break // ignore
			}
			v.(*tcpConnData).writeCh <- data
		case common.CmdClose:
			connID, err := common.UnpackBodyClose(tunr)
			if err != nil {
				log.Println("unpackBodyClose err", err)
				return
			}
			log.Printf("Read CmdClose, connID %d:%d,", tunID, connID)
			v, ok := connMap.Load(connID)
			if !ok {
				log.Printf("connID %d:%d not found", tunID, connID)
				break // ignore
			}
			close(v.(*tcpConnData).closeCh)
		}
	}
}
