package agent

import (
	"github.com/tutils/tnet/crypt"
	"github.com/tutils/tnet/tun"
)

// Options is the options of endpoint
type Options struct {
	tunServer       tun.Server // for normal mode
	tunClient       tun.Client // for reverse mode
	tunHandlerNewer AgentTunHandlerNewer
	tunCrypt        crypt.Crypt
}

// Option is option setter for agent
type Option func(opts *Options)

func newOptions(opts ...Option) *Options {
	opt := &Options{}
	for _, o := range opts {
		o(opt)
	}
	return opt
}

// WithTunServer sets tunnel server opt for normal mode
func WithTunServer(server tun.Server) Option {
	return func(opts *Options) {
		opts.tunServer = server
	}
}

// WithTunClient sets tunnel client opt for reverse mode
func WithTunClient(client tun.Client) Option {
	return func(opts *Options) {
		opts.tunClient = client
	}
}

type AgentTunHandlerNewer func(p *Agent) tun.Handler

// WithTunHandlerNewer sets tunnel handler newer opt
func WithTunHandlerNewer(newer AgentTunHandlerNewer) Option {
	return func(opts *Options) {
		opts.tunHandlerNewer = newer
	}
}

// WithTunCrypt sets tunnel crypt opt
func WithTunCrypt(crypt crypt.Crypt) Option {
	return func(opts *Options) {
		opts.tunCrypt = crypt
	}
}
