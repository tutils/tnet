package tcp

import (
	"context"
	"log"
	"net"
)

type ServerOptions struct {
	addr    string
	handler ServerHandler

	errorLog    *log.Logger
	baseContext func(net.Listener) context.Context
	connContext func(ctx context.Context, c net.Conn) context.Context
}

type ServerOption func(opts *ServerOptions)

func WithListenAddress(addr string) ServerOption {
	return func(opts *ServerOptions) {
		opts.addr = addr
	}
}

func WithServerHandler(h ServerHandler) ServerOption {
	return func(opts *ServerOptions) {
		opts.handler = h
	}
}

func WithErrorLogger(l *log.Logger) ServerOption {
	return func(opts *ServerOptions) {
		opts.errorLog = l
	}
}

func WithBaseContextFunc(f func(net.Listener) context.Context) ServerOption {
	return func(opts *ServerOptions) {
		opts.baseContext = f
	}
}

func WithConnContextFunc(f func(ctx context.Context, c net.Conn) context.Context) ServerOption {
	return func(opts *ServerOptions) {
		opts.connContext = f
	}
}
