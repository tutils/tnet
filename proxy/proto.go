package proxy

import (
	"encoding/binary"
	"errors"
	"io"
)

type Cmd int8

const (
	CmdConfig Cmd = iota
	CmdConnect
	CmdConnectResult
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

func packBodyConfig(w io.Writer, connectAddr string) error {
	if err := binary.Write(w, binary.BigEndian, int16(len(connectAddr))); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, []byte(connectAddr)); err != nil {
		return err
	}
	return nil
}

func unpackBodyConfig(r io.Reader) (connectAddr string, err error) {
	var n int16
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return connectAddr, err
	}
	b := make([]byte, n)
	if err := binary.Read(r, binary.BigEndian, b); err != nil {
		return connectAddr, err
	}
	connectAddr = string(b)
	return connectAddr, nil
}

func packBodyConnect(w io.Writer, connId int64) error {
	return binary.Write(w, binary.BigEndian, connId)
}

func unpackBodyConnect(r io.Reader) (connId int64, err error) {
	err = binary.Read(r, binary.BigEndian, &connId)
	return connId, err
}

func packBodyConnectResult(w io.Writer, connId int64, connectResult error) error {
	if err := binary.Write(w, binary.BigEndian, connId); err != nil {
		return err
	}
	if connectResult != nil {
		if err := binary.Write(w, binary.BigEndian, int16(len(connectResult.Error()))); err != nil {
			return err
		}
		if err := binary.Write(w, binary.BigEndian, []byte(connectResult.Error())); err != nil {
			return err
		}
	} else {
		if err := binary.Write(w, binary.BigEndian, int16(0)); err != nil {
			return err
		}
	}
	return nil
}

func unpackBodyConnectResult(r io.Reader) (connId int64, connectResult error, err error) {
	err = binary.Read(r, binary.BigEndian, &connId)
	var n int16
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return connId, connectResult, err
	}
	if n > 0 {
		b := make([]byte, n)
		if err := binary.Read(r, binary.BigEndian, b); err != nil {
			return connId, connectResult, err
		}
		connectResult = errors.New(string(b))
	}
	return connId, connectResult, nil
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

func unpackBodySend(r io.Reader) (connId int64, data []byte, err error) {
	if err := binary.Read(r, binary.BigEndian, &connId); err != nil {
		return connId, data, err
	}
	var n int64
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return connId, data, err
	}
	data = make([]byte, n)
	if err := binary.Read(r, binary.BigEndian, data); err != nil {
		return connId, data, err
	}
	return connId, data, nil
}

func packBodyClose(w io.Writer, connId int64) error {
	return binary.Write(w, binary.BigEndian, connId)
}

func unpackBodyClose(r io.Reader) (connId int64, err error) {
	err = binary.Read(r, binary.BigEndian, &connId)
	return connId, err
}
