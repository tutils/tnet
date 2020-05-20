package proxy

import (
	"github.com/tutils/tnet/crypt"
	"github.com/tutils/tnet/tun"
)

type EndpointOptions struct {
	tun      tun.Server
	tunCrypt crypt.Crypt
}

type EndpointOption func(opts *EndpointOptions)

var (
	DefaultTunServer = tun.NewServer()
)

func newEndpointOptions(opts ...EndpointOption) *EndpointOptions {
	opt := &EndpointOptions{}
	for _, o := range opts {
		o(opt)
	}

	if opt.tun == nil {
		opt.tun = DefaultTunServer
	}

	return opt
}

func WithTunServer(tun tun.Server) EndpointOption {
	return func(opts *EndpointOptions) {
		opts.tun = tun
	}
}

func WithTunServerCrypt(crypt crypt.Crypt) EndpointOption {
	return func(opts *EndpointOptions) {
		opts.tunCrypt = crypt
	}
}
