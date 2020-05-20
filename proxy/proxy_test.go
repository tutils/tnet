package proxy

import (
	"github.com/tutils/tnet/tun"
	"testing"
)

func TestNewProxy(t *testing.T) {
	p := NewProxy(
		WithTunClient(
			tun.NewClient(
				tun.WithConnectAddress("ws://127.0.0.1:8080/stream"),
				tun.WithClientHandler(NewTunClientHandler()),
			),
		),
		WithListenAddress(":56080"),
		WithConnectAddress("127.0.0.1:2888"),
		WithTunClientCrypt(DefaultTunCrypt),
	)
	p.DialAndServe()
}
