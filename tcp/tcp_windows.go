package tcp

import (
	"errors"
	"net"
)

func SetKeepAliveCount(conn *net.TCPConn, count int) (err error) {
	return errors.New("not supported")
}
