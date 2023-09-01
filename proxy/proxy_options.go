package proxy

import (
	"github.com/tutils/tnet/counter"
	"github.com/tutils/tnet/crypt"
	"github.com/tutils/tnet/crypt/xor"
	"github.com/tutils/tnet/tun"
)

// ProxyOptions is options of proxy
type ProxyOptions struct {
	tun             tun.Client
	tunCrypt        crypt.Crypt
	listenAddr      string
	connectAddr     string
	downloadCounter counter.Counter
	uploadCounter   counter.Counter
}

// ProxyOption is option setter for proxy
type ProxyOption func(opts *ProxyOptions)

// default proxy options
var (
	DefaultTunClient      = tun.NewClient()
	DefaultTunCrypt       = xor.NewCrypt(975135745)
	DefaultListenAddress  = ":"
	DefaultConnectAddress = ":3218"
)

func newProxyOptions(opts ...ProxyOption) *ProxyOptions {
	opt := &ProxyOptions{}
	for _, o := range opts {
		o(opt)
	}

	if opt.tun == nil {
		opt.tun = DefaultTunClient
	}
	if opt.listenAddr == "" {
		opt.listenAddr = DefaultListenAddress
	}
	if opt.connectAddr == "" {
		opt.connectAddr = DefaultConnectAddress
	}

	return opt
}

// WithTunClient sets tunnel client opt
func WithTunClient(tun tun.Client) ProxyOption {
	return func(opts *ProxyOptions) {
		opts.tun = tun
	}
}

// WithTunClientCrypt sets tunnel crypt opt
func WithTunClientCrypt(crypt crypt.Crypt) ProxyOption {
	return func(opts *ProxyOptions) {
		opts.tunCrypt = crypt
	}
}

// WithListenAddress sets local proxy listen address opt
func WithListenAddress(addr string) ProxyOption {
	return func(opts *ProxyOptions) {
		opts.listenAddr = addr
	}
}

// WithConnectAddress sets remote endpoint connect address opt
func WithConnectAddress(addr string) ProxyOption {
	return func(opts *ProxyOptions) {
		opts.connectAddr = addr
	}
}

// WithDownloadCounter sets download counter opt
func WithDownloadCounter(counter counter.Counter) ProxyOption {
	return func(opts *ProxyOptions) {
		opts.downloadCounter = counter
	}
}

// WithUploadCounter sets upload counter opt
func WithUploadCounter(counter counter.Counter) ProxyOption {
	return func(opts *ProxyOptions) {
		opts.uploadCounter = counter
	}
}
