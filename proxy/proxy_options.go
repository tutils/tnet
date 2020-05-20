package proxy

import "github.com/tutils/tnet/tun"

type ProxyOptions struct {
	tun tun.Client
}

type ProxyOption func(opts *ProxyOptions)

var DefaultTunClient = tun.NewClient()

func newProxyOptions(opts ...ProxyOption) *ProxyOptions {
	opt := &ProxyOptions{
		tun: DefaultTunClient,
	}

	for _, o := range opts {
		o(opt)
	}
	return opt
}

func WithTunClient(tun tun.Client) ProxyOption {
	return func(opts *ProxyOptions) {
		opts.tun = tun
	}
}
