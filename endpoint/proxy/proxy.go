package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"sync"
	"time"

	"github.com/tutils/tnet"
	"github.com/tutils/tnet/counter"
	"github.com/tutils/tnet/endpoint/common"
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

var units = []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
var log1024 = math.Log(1024.0)

func humanReadable(bytes uint64) string {
	base := 1024.0
	f := float64(bytes)
	if f < base {
		return fmt.Sprintf("%d %s", bytes, units[0])
	}
	exp := int(math.Log(f) / log1024)
	roundedSize := int64(f / math.Pow(base, float64(exp)))
	return fmt.Sprintf("%d %s", roundedSize, units[exp])
}

type tcpConnDataKey struct{}

type tcpConnData struct {
	connID       int64
	connectResCh chan error
	writeCh      chan []byte
	closeCh      chan struct{}
	writeDump    io.WriteCloser
	readDump     io.WriteCloser
}

// tcpHandler
type tcpHandler struct {
	tunw    io.Writer // SyncWriter
	tunID   int64
	connMap *sync.Map
	dumpDir string
}

// ServeTCP called from multiple goroutines
func (h *tcpHandler) ServeTCP(ctx context.Context, conn tcp.Conn) {
	// new proxy connection
	connr := conn.Reader()
	tunw := h.tunw
	connMap := h.connMap

	// conn_reader -> tun_writer
	connData := ctx.Value(tcpConnDataKey{}).(*tcpConnData)
	connID := connData.connID
	connMap.Store(connID, connData)
	log.Printf("new proxy connection, connID %d:%d", h.tunID, connID)
	defer log.Printf("proxy connection closed, connID %d:%d", h.tunID, connID)

	// create dump files if dumpDir is set
	if h.dumpDir != "" {
		dumpPath := fmt.Sprintf("%s/%d/%d", h.dumpDir, h.tunID, connID)
		if err := os.MkdirAll(dumpPath, 0755); err != nil {
			log.Printf("create dump dir err: %v", err)
			return
		}
		if writeDump, err := os.Create(fmt.Sprintf("%s/write.dmp", dumpPath)); err != nil {
			log.Printf("create write dump file err: %v", err)
			return
		} else {
			connData.writeDump = writeDump
		}
		if readDump, err := os.Create(fmt.Sprintf("%s/read.dmp", dumpPath)); err != nil {
			log.Printf("create read dump file err: %v", err)
			return
		} else {
			connData.readDump = readDump
		}
		defer func() {
			if connData.writeDump != nil {
				connData.writeDump.Close()
			}
			if connData.readDump != nil {
				connData.readDump.Close()
			}
		}()
	}

	tunwbuf := &bytes.Buffer{} // TODO: use pool
	if err := common.PackHeader(tunwbuf, common.CmdConnect); err != nil {
		log.Println("packHeader err", err)
		return
	}
	if err := common.PackBodyConnect(tunwbuf, connID); err != nil {
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
			if err := common.PackHeader(tunwbuf, common.CmdClose); err != nil {
				log.Println("packHeader err", err)
				break
			}
			if err := common.PackBodyClose(tunwbuf, connID); err != nil {
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
				if connData.writeDump != nil {
					if _, err := connData.writeDump.Write(data); err != nil {
						log.Printf("write dump file err: %v", err)
					}
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
				log.Printf("read conn abort: agent connection closed, connID %d:%d", h.tunID, connID)
			default:
				log.Printf("read conn err: %v, connID %d:%d", err, h.tunID, connID)
			}
			return
		}

		if connData.readDump != nil {
			if _, err := connData.readDump.Write(buf[:n]); err != nil {
				log.Printf("write dump file err: %v", err)
			}
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
		if cw, ok := tunw.(*counterWriter); ok {
			log.Printf("Write CmdSend, connID %d:%d, %d bytes, upload %s/s", h.tunID, connID, tunwbuf.Len(), humanReadable(uint64(cw.c.IncreaceRatePerSec())))
		} else {
			log.Printf("Write CmdSend, connID %d:%d, %d bytes", h.tunID, connID, tunwbuf.Len())
		}
		if _, err := tunw.Write(tunwbuf.Bytes()); err != nil {
			log.Println("write tun err", err)
			return
		}
	}
}

var _ tun.Handler = (*proxyTunHandler)(nil)

type proxyTunHandler struct {
	p *Proxy
}

func NewTCPProxyTunHandler(p *Proxy) tun.Handler {
	return &proxyTunHandler{
		p: p,
	}
}

// ServeTun implements tun.Handler.
func (h *proxyTunHandler) ServeTun(ctx context.Context, r io.Reader, w io.Writer) {
	log.Println("new tun connection")
	defer log.Println("tun connection closed")

	opts := &h.p.opts
	// tcp tunnel has been setup
	var tunw io.Writer
	if crypt := opts.tunCrypt; crypt != nil {
		tunw = crypt.NewEncoder(w)
	} else {
		tunw = w
	}
	tunw = tnet.NewSyncWriter(tunw)
	if counter := opts.uploadCounter; counter != nil {
		tunw = &counterWriter{w: tunw, c: counter}
	}

	var tunr io.Reader
	if crypt := opts.tunCrypt; crypt != nil {
		tunr = crypt.NewDecoder(r)
	} else {
		tunr = r
	}
	if counter := opts.downloadCounter; counter != nil {
		tunr = &counterReader{r: tunr, c: counter}
	}

	var connMap sync.Map

	// send config: connect to
	buf := &bytes.Buffer{} // TODO: use pool
	if err := common.PackHeader(buf, common.CmdConfig); err != nil {
		log.Println("packHeader err", err)
		return
	}
	if err := common.PackBodyConfig(buf, opts.connectAddr); err != nil {
		log.Println("packBodyConfig err", err)
		return
	}
	log.Printf("Write CmdConfig, connectAddr %s", opts.connectAddr)
	if _, err := tunw.Write(buf.Bytes()); err != nil {
		log.Println("write tun err", err)
		return
	}

	// sync tunID
	isServer := opts.tunServer != nil
	tunID, err := common.SyncTunID(ctx, isServer, tunr, tunw)
	if err != nil {
		return
	}

	tcph := &tcpHandler{
		tunw:    tunw,
		tunID:   tunID,
		connMap: &connMap,
		dumpDir: opts.dumpDir,
	}

	var connID int64

	s := tcp.NewServer(
		tcp.WithListenAddress(opts.listenAddr),
		tcp.WithServerHandler(tcp.NewRawTCPConnHandler(tcph)),
		tcp.WithServerConnContextFunc(func(ctx context.Context, c net.Conn) context.Context {
			connID++
			data := &tcpConnData{
				connID:       connID,
				connectResCh: make(chan error, 1),
				writeCh:      make(chan []byte, 1<<8),
				closeCh:      make(chan struct{}),
			}
			return context.WithValue(ctx, tcpConnDataKey{}, data)
		}),
		tcp.WithServerKeepAlivePeriod(time.Second*15),
		tcp.WithServerKeepAliveCount(3),
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.ListenAndServe()
	}()
	defer s.Shutdown(context.Background())

	log.Printf("tcp server listen on %s", opts.listenAddr)

	// tun_reader -> conn_writer
	for {
		select {
		case <-errCh:
			return
		default:
		}

		cmd, err := common.UnpackHeader(tunr)
		if err != nil {
			log.Println("unpackHeader err", err)
			return
		}
		switch cmd {
		case common.CmdConnectResult:
			connID, connectResult, err := common.UnpackBodyConnectResult(tunr)
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
			v.(*tcpConnData).connectResCh <- connectResult

		case common.CmdSend:
			connID, data, err := common.UnpackBodySend(tunr)
			if cr, ok := tunr.(*counterReader); ok {
				log.Printf("Read CmdSend, connID %d:%d, %d bytes, download %s/s", tunID, connID, len(data), humanReadable(uint64(cr.c.IncreaceRatePerSec())))
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
			v.(*tcpConnData).writeCh <- data

		case common.CmdClose:
			connID, err := common.UnpackBodyClose(tunr)
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
			close(v.(*tcpConnData).closeCh)
		}
	}
}

// Proxy for proxying remote tcp server to local address
type Proxy struct {
	opts Options
}

// New create a new proxy
func New(opts ...Option) *Proxy {
	opt := newOptions(opts...)
	return &Proxy{
		opts: *opt,
	}
}

// Serve starts proxy
func (p *Proxy) Serve() error {
	if tunServer := p.opts.tunServer; tunServer != nil {
		log.Println("start tun server (reverse mode)")
		defer log.Println("tun server exit")
		h := p.opts.tunHandlerNewer(p)
		return tunServer.ListenAndServe(h)
	}

	if tunClient := p.opts.tunClient; tunClient != nil {
		log.Println("start tun client")
		defer log.Println("tun client exit")
		h := p.opts.tunHandlerNewer(p)
		return tunClient.DialAndServe(h)
	}

	return fmt.Errorf("neither tunnel client nor server is configured")
}
