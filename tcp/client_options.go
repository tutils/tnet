package tcp

import (
	"context"
	"net"
)

type ClientOptions struct {
	addr    string
	handler ConnHandler

	connContext  func(ctx context.Context, c net.Conn) context.Context
	errorLogFunc func(fmt string, args ...interface{})
}

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

func WithConnectAddress(addr string) ClientOption {
	return func(opts *ClientOptions) {
		opts.addr = addr
	}
}

func WithClientHandler(h ConnHandler) ClientOption {
	return func(opts *ClientOptions) {
		opts.handler = h
	}
}

func WithClientConnContextFunc(f func(ctx context.Context, c net.Conn) context.Context) ClientOption {
	return func(opts *ClientOptions) {
		opts.connContext = f
	}
}

func WithClientErrorLogFunc(errorLogFunc func(fmt string, args ...interface{})) ClientOption {
	return func(opts *ClientOptions) {
		opts.errorLogFunc = errorLogFunc
	}
}
