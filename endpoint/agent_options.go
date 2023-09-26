package endpoint

import (
	"github.com/tutils/tnet/crypt"
	"github.com/tutils/tnet/tun"
	"github.com/tutils/tnet/tun/websocket"
)

// AgentOptions is the options of agent
type AgentOptions struct {
	tun      tun.Server
	tunCrypt crypt.Crypt
}

// AgentOption is option setter for agent
type AgentOption func(opts *AgentOptions)

// default agent options
var (
	DefaultTunServer = websocket.NewServer()
)

func newAgentOptions(opts ...AgentOption) *AgentOptions {
	opt := &AgentOptions{}
	for _, o := range opts {
		o(opt)
	}

	if opt.tun == nil {
		opt.tun = DefaultTunServer
	}

	return opt
}

// WithTunServer sets tunnel server opt
func WithTunServer(tun tun.Server) AgentOption {
	return func(opts *AgentOptions) {
		opts.tun = tun
	}
}

// WithTunServerCrypt sets tunnel crypt opt
func WithTunServerCrypt(crypt crypt.Crypt) AgentOption {
	return func(opts *AgentOptions) {
		opts.tunCrypt = crypt
	}
}
