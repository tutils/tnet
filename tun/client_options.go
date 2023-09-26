package tun

// ClientOptions is client options
type ClientOptions struct {
	Address string
	Handler Handler
}

// ClientOption is option setter for client
type ClientOption func(*ClientOptions)

// default client options
var (
	DefaultConnectAddress = "ws://127.0.0.1:8080/stream"
)

func NewClientOptions(opts ...ClientOption) *ClientOptions {
	opt := &ClientOptions{}
	for _, o := range opts {
		o(opt)
	}

	if opt.Address == "" {
		opt.Address = DefaultConnectAddress
	}

	return opt
}

// WithConnectAddress sets client connect address opt
func WithConnectAddress(addr string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Address = addr
	}
}

// WithClientHandler sets tunnel handler opt
func WithClientHandler(h Handler) ClientOption {
	return func(opts *ClientOptions) {
		opts.Handler = h
	}
}
