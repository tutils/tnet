package tcp

import (
	"context"
	"net"
)

type ServerOptions struct {
	addr    string
	handler ConnHandler

	baseContext  func(net.Listener) context.Context
	connContext  func(ctx context.Context, c net.Conn) context.Context
	errorLogFunc func(fmt string, args ...interface{})
}

type ServerOption func(opts *ServerOptions)

func newServerOptions(opts ...ServerOption) *ServerOptions {
	opt := &ServerOptions{}
	for _, o := range opts {
		o(opt)
	}

	if opt.addr == "" {
		opt.addr = ":"
	}
	if opt.handler == nil {
		opt.handler = DefaultConnHandler
	}
	if opt.errorLogFunc == nil {
		opt.errorLogFunc = DefaultErrorLogFunc
	}

	return opt
}

func WithListenAddress(addr string) ServerOption {
	return func(opts *ServerOptions) {
		opts.addr = addr
	}
}

func WithServerHandler(h ConnHandler) ServerOption {
	return func(opts *ServerOptions) {
		opts.handler = h
	}
}

func WithServerBaseContextFunc(f func(net.Listener) context.Context) ServerOption {
	return func(opts *ServerOptions) {
		opts.baseContext = f
	}
}

func WithServerConnContextFunc(f func(ctx context.Context, c net.Conn) context.Context) ServerOption {
	return func(opts *ServerOptions) {
		opts.connContext = f
	}
}

func WithServerErrorLogFunc(errorLogFunc func(fmt string, args ...interface{})) ServerOption {
	return func(opts *ServerOptions) {
		opts.errorLogFunc = errorLogFunc
	}
}
