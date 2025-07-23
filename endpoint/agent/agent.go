package agent

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/tutils/tnet"
	"github.com/tutils/tnet/endpoint/common"
	"github.com/tutils/tnet/tun"
)

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
		h := a.opts.tunHandlerNewer(a)
		return tunClient.DialAndServe(h)
	}

	if tunServer := a.opts.tunServer; tunServer != nil {
		log.Println("start tun server")
		defer log.Println("tun server exit")
		h := a.opts.tunHandlerNewer(a)
		return tunServer.ListenAndServe(h)
	}

	return fmt.Errorf("neither tunnel client nor server is configured")
}

var _ tun.Handler = (*agentTunHandler)(nil)

type agentTunHandler struct {
	a *Agent
}

func NewTCPAgentTunHandler(a *Agent) tun.Handler {
	return &agentTunHandler{
		a: a,
	}
}

// ServeTun implements tun.Handler.
func (h *agentTunHandler) ServeTun(ctx context.Context, r io.Reader, w io.Writer) {
	log.Println("new tun connection")
	defer log.Println("tun connection closed")

	opts := &h.a.opts

	// new tun connection
	var tunr io.Reader
	if crypt := opts.tunCrypt; crypt != nil {
		tunr = crypt.NewDecoder(r)
	} else {
		tunr = r
	}

	var tunw io.Writer
	if crypt := opts.tunCrypt; crypt != nil {
		tunw = crypt.NewEncoder(w)
	} else {
		tunw = w
	}
	tunw = tnet.NewSyncWriter(tunw)

	// sync tunID
	isServer := opts.tunServer != nil
	tunID, err := common.SyncTunID(ctx, isServer, tunr, tunw)
	if err != nil {
		return
	}

	// recv config: connect to
	cmd, err := common.UnpackHeader(tunr)
	if err != nil {
		log.Println("unpackHeader err", err)
		return
	}

	switch cmd {
	case common.CmdConfig:
		h.agentTCP(ctx, tunID, tunr, tunw)
	case common.CmdConnectPTY:
		if !opts.enabledExecute {
			log.Println("CmdConnectPTY not enabled", err)
			return
		}
		h.agentPTY(ctx, tunID, tunr, tunw)
	default:
	
		log.Println("invalid cmd")
		return
	}
}
