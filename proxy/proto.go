package proxy

import (
	"encoding/binary"
	"errors"
	"io"
)

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

func unpackBodySend(r io.Reader, f func(connId int64) io.Writer) (connId int64, err error) {
	if err := binary.Read(r, binary.BigEndian, &connId); err != nil {
		return connId, err
	}
	var n int64
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return connId, err
	}
	w := f(connId)
	if w == nil {
		return connId, errors.New("nil writer")
	}
	if _, err := io.CopyN(w, r, n); err != nil {
		return connId, err
	}
	return connId, nil
}

func packBodyClose(w io.Writer, connId int64) error {
	return binary.Write(w, binary.BigEndian, connId)
}

func unpackBodyClose(r io.Reader) (connId int64, err error) {
	err = binary.Read(r, binary.BigEndian, &connId)
	return connId, err
}
