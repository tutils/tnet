package tun

// Server is tunnel server
type Server interface {
	ListenAndServe(h Handler) error
}

// default server
var (
	NewServer = newWsServer
)
