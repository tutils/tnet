package tun

// Client is tunnel client
type Client interface {
	Handler() Handler
	DialAndServe() error
}
