package tun

import "context"

// Client is tunnel client
type Client interface {
	DialAndServe(ctx context.Context, h Handler) error
}

// default client
var (
	NewClient = newWsClient
)
