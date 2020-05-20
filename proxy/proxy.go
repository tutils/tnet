package proxy

import (
	"github.com/tutils/tnet/tcp"
)

type Proxy struct {
	tcpSrv *tcp.Server
}

func (p *Proxy) DialTunnel() error {
	return nil
}

func NewProxy() *Proxy {
	return &Proxy{}
}
