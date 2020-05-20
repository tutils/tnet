package proxy

import (
	"bufio"
	"context"
	"encoding/binary"
	"github.com/tutils/tnet/crypt/xor"
	"github.com/tutils/tnet/tcp"
	"github.com/tutils/tnet/tun"
	"io"
	"net"
	"testing"
)

type connIdKey struct {}

var connId int64 = 0

func connIdGen() int64 {
	connId++
	return connId
}

type Cmd int8

const (
	CmdConnect Cmd = iota
	CmdSend
	CmdClose
)

func packHeader(w io.Writer, cmd Cmd) error {
	return binary.Write(w, binary.BigEndian, cmd)
}

func unpackHeader(r io.Reader) (cmd Cmd, err error) {
	err = binary.Read(r, binary.BigEndian, &cmd)
	return cmd, err
}

func packBodyConnect(w io.Writer, connId int64) error {
	return binary.Write(w, binary.BigEndian, connId)
}

func unpackBodyConnect(r io.Reader) (connId int64, err error) {
	err = binary.Read(r, binary.BigEndian, &connId)
	return connId, err
}

func packBodySend(w io.Writer, connId int64, data []byte) error {
	if err := binary.Write(w, binary.BigEndian, connId); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, int64(len(data))); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, data); err != nil {
		return err
	}
	return nil
}

func unpackBodySend(r io.Reader, w io.Writer) error {
	if err := binary.Read(r, binary.BigEndian, &connId); err != nil {
		return err
	}
	var n int64
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return err
	}
	if _, err := io.CopyN(w, r, n); err != nil {
		return err
	}
	return nil
}

func packBodyClose(w io.Writer, connId int64) error {
	return binary.Write(w, binary.BigEndian, connId)
}

func unpackBodyClose(r io.Reader) (connId int64, err error) {
	err = binary.Read(r, binary.BigEndian, &connId)
	return connId, err
}

type tcphandler struct {
	t *testing.T
	s *tcp.Server
	tunw io.Writer
}

func (h *tcphandler) ServeTCP(ctx context.Context, conn *tcp.Conn) {
	// new proxy connection
	cr := conn.Reader()
	tw := bufio.NewWriter(h.tunw)

	buf := make([]byte, 4<<10)

	// conn_reader -> tun_writer
	connId := ctx.Value(connIdKey{}).(int64)

	if err := packHeader(tw, CmdConnect); err != nil {
		return
	}
	if err := packBodyConnect(tw, connId); err != nil {
		return
	}
	if err := tw.Flush(); err != nil {
		return
	}

	for {
		n, err := cr.Read(buf)
		if err != nil {
			return
		}
		if err := packHeader(tw, CmdSend); err != nil {
			return
		}
		if err := packBodySend(tw, connId, buf[:n]); err != nil {
			return
		}
		if err := tw.Flush(); err != nil {
			return
		}
	}
}

type tunchandler struct {
	t *testing.T
}
var cr = xor.NewCrypt(812734)

func (h *tunchandler) ServeTun(r io.Reader, w io.Writer) {
	// tcp tunnel has been setup
	tcph := &tcphandler{
		t: h.t,
		tunw: cr.NewEncoder(w),
	}

	s := tcp.NewServer(
		tcp.WithListenAddress(":56081"),
		tcp.WithServerHandler(tcp.NewServerHandler(tcph)),
		tcp.WithConnContextFunc(func(ctx context.Context, c net.Conn) context.Context {
			return context.WithValue(ctx, connIdKey{}, connIdGen())
		}),
	)
	tcph.s = s
	s.ListenAndServe()
}

func TestNewProxy(t *testing.T) {
	h := &tunchandler{t: t}
	tc := tun.NewClient(
		tun.WithConnectAddress("ws://127.0.0.1:8080/stream"),
		tun.WithClientHandler(h),
	)
	tc.DialAndServe()
}
