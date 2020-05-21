package tcp

import (
	"context"
	"net"
)

// ClientOptions is options of tcp client
type ClientOptions struct {
	addr    string
	handler ConnHandler

	connContext  func(ctx context.Context, c net.Conn) context.Context
	errorLogFunc func(fmt string, args ...interface{})
}

// ClientOption is option setter for tcp client
type ClientOption func(opts *ClientOptions)

func newClientOptions(opts ...ClientOption) *ClientOptions {
	opt := &ClientOptions{}
	for _, o := range opts {
		o(opt)
	}

	if opt.handler == nil {
		opt.handler = DefaultConnHandler
	}
	if opt.errorLogFunc == nil {
		opt.errorLogFunc = DefaultErrorLogFunc
	}

	return opt
}

// WithConnectAddress sets connect address opt
func WithConnectAddress(addr string) ClientOption {
	return func(opts *ClientOptions) {
		opts.addr = addr
	}
}

// WithClientHandler sets client handler opt
func WithClientHandler(h ConnHandler) ClientOption {
	return func(opts *ClientOptions) {
		opts.handler = h
	}
}

// WithClientConnContextFunc sets connection context hook function opt
func WithClientConnContextFunc(f func(ctx context.Context, c net.Conn) context.Context) ClientOption {
	return func(opts *ClientOptions) {
		opts.connContext = f
	}
}

// WithClientErrorLogFunc sets error log function opt
func WithClientErrorLogFunc(errorLogFunc func(fmt string, args ...interface{})) ClientOption {
	return func(opts *ClientOptions) {
		opts.errorLogFunc = errorLogFunc
	}
}
