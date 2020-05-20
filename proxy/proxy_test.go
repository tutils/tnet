package proxy

import (
	"github.com/tutils/tnet/tun"
	"testing"
)

func TestNewProxy(t *testing.T) {
	h := &tunHandler{t: t}
	tc := tun.NewClient(
		tun.WithConnectAddress("ws://127.0.0.1:8080/stream"),
		tun.WithClientHandler(h),
	)
	tc.DialAndServe()
}
