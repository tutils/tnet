package tun

type ServerOptions struct {
	addr    string
	handler Handler
}

type ServerOption func(*ServerOptions)

var (
	DefaultListenAddress = "ws://0.0.0.0:8080/stream"
)

func newServerOptions(opts ...ServerOption) *ServerOptions {
	opt := &ServerOptions{}
	for _, o := range opts {
		o(opt)
	}

	if opt.addr == "" {
		opt.addr = DefaultListenAddress
	}

	return opt
}

func WithListenAddress(addr string) ServerOption {
	return func(opts *ServerOptions) {
		opts.addr = addr
	}
}

func WithServerHandler(h Handler) ServerOption {
	return func(opts *ServerOptions) {
		opts.handler = h
	}
}
