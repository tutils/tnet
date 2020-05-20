package tun

type Client interface {
	Handler() Handler
	DialAndServe() error
}

var (
	NewClient = newWsClient
)
