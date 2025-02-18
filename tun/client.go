package tun

// Client is tunnel client
type Client interface {
	DialAndServe(h Handler) error
}

// default client
var (
	NewClient = newWsClient
)
