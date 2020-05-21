package proxy

import (
	"github.com/tutils/tnet/crypt/xor"
	"github.com/tutils/tnet/tun"
	"testing"
)

func TestNewProxy(t *testing.T) {
	p := NewProxy(
		WithTunClient(
			tun.NewClient(
				tun.WithConnectAddress("ws://tvps.tutils.com:8080/stream"),
				tun.WithClientHandler(NewTunClientHandler()),
			),
		),
		WithListenAddress(":56080"),
		WithConnectAddress("127.0.0.1:53128"),
		WithTunClientCrypt(xor.NewCrypt(816559)),
	)
	p.DialAndServe()
}
