package tun

type ClientOptions struct {
	addr    string
	handler Handler
}

type ClientOption func(*ClientOptions)

var DefaultConnectAddress = "ws://127.0.0.1:8080/stream"

func newClientOptions(opts ...ClientOption) *ClientOptions {
	opt := &ClientOptions{
		addr: DefaultConnectAddress,
	}

	for _, o := range opts {
		o(opt)
	}
	return opt
}

func WithConnectAddress(addr string) ClientOption {
	return func(opts *ClientOptions) {
		opts.addr = addr
	}
}

func WithClientHandler(h Handler) ClientOption {
	return func(opts *ClientOptions) {
		opts.handler = h
	}
}
