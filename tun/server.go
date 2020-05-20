package tun

type Server interface {
	Handler() Handler
	ListenAndServe() error
}

var (
	NewServer = newWsServer
)
