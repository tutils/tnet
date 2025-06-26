package tcp

import (
	"errors"
	"net"
)

// SetKeepAliveCount set the TCP_KEEPCNT option
func SetKeepAliveCount(conn *net.TCPConn, count int) (err error) {
	return errors.New("not supported")
}
