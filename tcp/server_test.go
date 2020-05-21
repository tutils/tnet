package tcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"testing"
	"time"
)

type handler struct {
	t *testing.T
}

func (h *handler) ServeTCP(ctx context.Context, conn Conn) {
	bufr := conn.BufferReader()
	bufw := conn.BufferWriter()

	for {
		ln, _, err := bufr.ReadLine()
		if err != nil {
			return
		}

		bufw.WriteString("OK\n")
		bufw.Flush()
		h.t.Logf("%s", string(ln))
	}
}

func TestListenAndServe(t *testing.T) {
	h := &handler{t: t}
	ch := NewRawTCPConnHandler(h)
	server := NewServer(
		WithListenAddress(":8080"),
		WithServerHandler(ch),
	)
	go server.ListenAndServe()
	time.Sleep(time.Second * 5)
	t.Logf("shutdown ... %#v", server.activeConn)
	server.Shutdown(context.Background())
	t.Logf("done")
	time.Sleep(time.Second * 5)
	t.Logf("%#v", server.activeConn)
}

func TestScanner(t *testing.T) {
	var typ int16 = 10
	data := []byte("abcdefg")
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.BigEndian, typ)
	t.Log(buf.Len())
	binary.Write(buf, binary.BigEndian, int64(len(data)))
	t.Log(buf.Len())
	binary.Write(buf, binary.BigEndian, data)
	t.Log(buf.Len())

	data = []byte("12341234")
	binary.Write(buf, binary.BigEndian, typ)
	binary.Write(buf, binary.BigEndian, int64(len(data)))
	binary.Write(buf, binary.BigEndian, data)

	data = []byte("a7sdfa6ds8fu")
	binary.Write(buf, binary.BigEndian, typ)
	binary.Write(buf, binary.BigEndian, int64(len(data)))

	s := bufio.NewScanner(buf)
	s.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if len(data) < 10 {
			return 0, nil, nil
		}
		var l int64
		binary.Read(bytes.NewReader(data[2:]), binary.BigEndian, &l)
		if int64(len(data[10:])) >= l {
			return 10 + int(l), data[10 : 10+l], nil
		}
		if atEOF {
			return 0, nil, bufio.ErrFinalToken
		}
		return 0, nil, nil
	})
	for s.Scan() {
		t.Log(string(s.Bytes()))
	}

	binary.Write(buf, binary.BigEndian, data)
	for s.Scan() {
		t.Log(string(s.Bytes()))
	}
}
