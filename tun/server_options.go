package tun

// ServerOptions is server options
type ServerOptions struct {
	Address string
	Handler Handler
}

// ServerOption is option setter for server
type ServerOption func(*ServerOptions)

// default server options
var (
	DefaultListenAddress = "ws://0.0.0.0:8080/stream"
)

func NewServerOptions(opts ...ServerOption) *ServerOptions {
	opt := &ServerOptions{}
	for _, o := range opts {
		o(opt)
	}

	if opt.Address == "" {
		opt.Address = DefaultListenAddress
	}

	return opt
}

// WithListenAddress sets server listen address opt
func WithListenAddress(addr string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Address = addr
	}
}

// WithServerHandler sets tunnel handler opt
func WithServerHandler(h Handler) ServerOption {
	return func(opts *ServerOptions) {
		opts.Handler = h
	}
}
