package tcp

import (
	"context"
	"net"
	"time"
)

// ServerOptions is server options
type ServerOptions struct {
	addr    string
	handler ConnHandler

	baseContext     func(net.Listener) context.Context
	connContext     func(ctx context.Context, c net.Conn) context.Context
	errorLogFunc    func(fmt string, args ...interface{})
	keepAlivePeriod time.Duration
	keepAliveCount  int
}

// ServerOption is option setter for server
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

// WithListenAddress sets listen address opt
func WithListenAddress(addr string) ServerOption {
	return func(opts *ServerOptions) {
		opts.addr = addr
	}
}

// WithServerHandler sets server handler opt
func WithServerHandler(h ConnHandler) ServerOption {
	return func(opts *ServerOptions) {
		opts.handler = h
	}
}

// WithServerBaseContextFunc sets server base context hook funcion opt
func WithServerBaseContextFunc(f func(net.Listener) context.Context) ServerOption {
	return func(opts *ServerOptions) {
		opts.baseContext = f
	}
}

// WithServerConnContextFunc sets new connection context hook funcion opt
func WithServerConnContextFunc(f func(ctx context.Context, c net.Conn) context.Context) ServerOption {
	return func(opts *ServerOptions) {
		opts.connContext = f
	}
}

// WithServerErrorLogFunc sets error log function opt
func WithServerErrorLogFunc(errorLogFunc func(fmt string, args ...interface{})) ServerOption {
	return func(opts *ServerOptions) {
		opts.errorLogFunc = errorLogFunc
	}
}

// WithServerKeepAlivePeriod sets tcp keepalive period opt
func WithServerKeepAlivePeriod(period time.Duration) ServerOption {
	return func(opts *ServerOptions) {
		opts.keepAlivePeriod = period
	}
}

// WithServerKeepAliveCount sets tcp keepalive count opt
func WithServerKeepAliveCount(count int) ServerOption {
	return func(opts *ServerOptions) {
		opts.keepAliveCount = count
	}
}
