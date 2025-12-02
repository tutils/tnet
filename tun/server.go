package tun

import "context"

// Server is tunnel server
type Server interface {
	ListenAndServe(ctx context.Context, h Handler) error
}

// default server
var (
	NewServer = newWsServer
)
