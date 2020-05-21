package tun

// client options
type ClientOptions struct {
	addr    string
	handler Handler
}

// client option
type ClientOption func(*ClientOptions)

// default client options
var (
	DefaultConnectAddress = "ws://127.0.0.1:8080/stream"
)

func newClientOptions(opts ...ClientOption) *ClientOptions {
	opt := &ClientOptions{}
	for _, o := range opts {
		o(opt)
	}

	if opt.addr == "" {
		opt.addr = DefaultConnectAddress
	}

	return opt
}

// client connect address opt
func WithConnectAddress(addr string) ClientOption {
	return func(opts *ClientOptions) {
		opts.addr = addr
	}
}

// tunnel handler opt
func WithClientHandler(h Handler) ClientOption {
	return func(opts *ClientOptions) {
		opts.handler = h
	}
}
