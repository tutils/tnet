package proxy

import (
	"github.com/tutils/tnet/tun"
	"testing"
)

func TestNewEndpoint(t *testing.T) {
	e := NewEndpoint(
		WithTunServer(
			tun.NewServer(
				tun.WithListenAddress("ws://0.0.0.0:8080/stream"),
				tun.WithServerHandler(NewTunServerHandler()),
			),
		),
		WithTunServerCrypt(DefaultTunCrypt),
	)
	e.ListenAndServe()
}
