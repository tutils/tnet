package tcp

import (
	"net"
	"syscall"
)

func SetKeepAliveCount(conn *net.TCPConn, count int) (err error) {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return err
	}

	rawConn.Control(func(fdPtr uintptr) {
		// got socket file descriptor. Setting parameters.
		fd := int(fdPtr)
		//Number of probes.
		err = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPCNT, count)
	})

	return err
}
