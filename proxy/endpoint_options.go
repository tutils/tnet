package proxy

import (
	"github.com/tutils/tnet/crypt"
	"github.com/tutils/tnet/tun"
)

// EndpointOptions is the options of endpoint
type EndpointOptions struct {
	tun      tun.Server
	tunCrypt crypt.Crypt
}

// EndpointOption is option setter for endpoint
type EndpointOption func(opts *EndpointOptions)

// default endpoint options
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

// WithTunServer sets tunnel server opt
func WithTunServer(tun tun.Server) EndpointOption {
	return func(opts *EndpointOptions) {
		opts.tun = tun
	}
}

// WithTunServerCrypt sets tunnel crypt opt
func WithTunServerCrypt(crypt crypt.Crypt) EndpointOption {
	return func(opts *EndpointOptions) {
		opts.tunCrypt = crypt
	}
}
