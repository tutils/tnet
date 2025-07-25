package proxy

import (
	"github.com/tutils/tnet/counter"
	"github.com/tutils/tnet/crypt"
	"github.com/tutils/tnet/tun"
)

// Options is options of proxy
type Options struct {
	tunClient       tun.Client // for normal mode
	tunServer       tun.Server // for reverse mode
	tunHandlerNewer ProxyTunHandlerNewer
	tunCrypt        crypt.Crypt
	listenAddr      string
	connectAddr     string
	executeArgs     []string
	rawPTYMode      bool
	downloadCounter counter.Counter
	uploadCounter   counter.Counter
	dumpDir         string
}

// Option is option setter for proxy
type Option func(opts *Options)

// default proxy options
func newOptions(opts ...Option) *Options {
	opt := &Options{}
	for _, o := range opts {
		o(opt)
	}

	return opt
}

// WithTunClient sets tunnel client opt
func WithTunClient(client tun.Client) Option {
	return func(opts *Options) {
		opts.tunClient = client
	}
}

// WithTunServer sets tunnel server opt for reverse mode
func WithTunServer(server tun.Server) Option {
	return func(opts *Options) {
		opts.tunServer = server
	}
}

type ProxyTunHandlerNewer func(p *Proxy) tun.Handler

// WithTunHandlerNewer sets tunnel handler newer opt
func WithTunHandlerNewer(newer ProxyTunHandlerNewer) Option {
	return func(opts *Options) {
		opts.tunHandlerNewer = newer
	}
}

// WithTunCrypt sets tunnel crypt opt for normal mode
func WithTunCrypt(crypt crypt.Crypt) Option {
	return func(opts *Options) {
		opts.tunCrypt = crypt
	}
}

// WithListenAddress sets local proxy listen address opt
func WithListenAddress(addr string) Option {
	return func(opts *Options) {
		opts.listenAddr = addr
	}
}

// WithConnectAddress sets remote agent connect address opt
func WithConnectAddress(addr string) Option {
	return func(opts *Options) {
		opts.connectAddr = addr
	}
}

// WithConnectPTY sets remote agent connect pty opt
func WithConnectPTY(args []string) Option {
	return func(opts *Options) {
		opts.executeArgs = args
	}
}

// WithRawPTYMode sets raw pty mode opt
func WithRawPTYMode(rawPTYMode bool) Option {
	return func(opts *Options) {
		opts.rawPTYMode = rawPTYMode
	}
}

// WithDownloadCounter sets download counter opt
func WithDownloadCounter(counter counter.Counter) Option {
	return func(opts *Options) {
		opts.downloadCounter = counter
	}
}

// WithUploadCounter sets upload counter opt
func WithUploadCounter(counter counter.Counter) Option {
	return func(opts *Options) {
		opts.uploadCounter = counter
	}
}

// WithDumpDir sets dump directory opt
func WithDumpDir(dir string) Option {
	return func(opts *Options) {
		opts.dumpDir = dir
	}
}
