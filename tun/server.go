package tun

// Server is tunnel server
type Server interface {
	Handler() Handler
	ListenAndServe() error
}
