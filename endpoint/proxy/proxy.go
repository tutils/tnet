package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/tutils/tnet"
	"github.com/tutils/tnet/endpoint/common"
	"github.com/tutils/tnet/tun"
)

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

var _ tun.Handler = (*proxyTunHandler)(nil)

type proxyTunHandler struct {
	p *Proxy
}

func NewProxyTunHandler(p *Proxy) tun.Handler {
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
	var tunr io.Reader
	if crypt := opts.tunCrypt; crypt != nil {
		tunr = crypt.NewDecoder(r)
	} else {
		tunr = r
	}
	if counter := opts.downloadCounter; counter != nil {
		tunr = &counterReader{r: tunr, c: counter}
	}

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

	// sync tunID
	isServer := opts.tunServer != nil
	tunID, err := common.SyncTunID(ctx, isServer, tunr, tunw)
	if err != nil {
		return
	}

	if len(opts.listenAddr) > 0 {
		h.proxyTCP(ctx, tunID, tunr, tunw)
	} else if len(opts.executeArgs) > 0 {
		os.Exit(h.proxyPTY(ctx, tunID, tunr, tunw))
	} else {
		log.Println("invalid options")
	}
}
