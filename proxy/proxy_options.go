package proxy

import (
	"github.com/tutils/tnet/crypt"
	"github.com/tutils/tnet/crypt/xor"
	"github.com/tutils/tnet/tun"
)

type ProxyOptions struct {
	tun         tun.Client
	tunCrypt    crypt.Crypt
	listenAddr  string
	connectAddr string
}

type ProxyOption func(opts *ProxyOptions)

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

func WithTunClient(tun tun.Client) ProxyOption {
	return func(opts *ProxyOptions) {
		opts.tun = tun
	}
}

func WithTunClientCrypt(crypt crypt.Crypt) ProxyOption {
	return func(opts *ProxyOptions) {
		opts.tunCrypt = crypt
	}
}

func WithListenAddress(addr string) ProxyOption {
	return func(opts *ProxyOptions) {
		opts.listenAddr = addr
	}
}

func WithConnectAddress(addr string) ProxyOption {
	return func(opts *ProxyOptions) {
		opts.connectAddr = addr
	}
}
