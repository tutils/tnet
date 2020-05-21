package tun

// ServerOptions is server options
type ServerOptions struct {
	addr    string
	handler Handler
}

// ServerOption is option setter for server
type ServerOption func(*ServerOptions)

// default server options
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

// WithListenAddress sets server listen address opt
func WithListenAddress(addr string) ServerOption {
	return func(opts *ServerOptions) {
		opts.addr = addr
	}
}

// WithServerHandler sets tunnel handler opt
func WithServerHandler(h Handler) ServerOption {
	return func(opts *ServerOptions) {
		opts.handler = h
	}
}
