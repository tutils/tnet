package common

import (
	"encoding/binary"
	"errors"
	"io"
)

// Cmd is command code of tunnel packet
type Cmd int8

// command values
const (
	CmdConfig Cmd = iota
	CmdTunID
	CmdConnect
	CmdConnectResult
	CmdSend
	CmdClose
)

func PackHeader(w io.Writer, cmd Cmd) error {
	return binary.Write(w, binary.BigEndian, cmd)
}

func UnpackHeader(r io.Reader) (cmd Cmd, err error) {
	err = binary.Read(r, binary.BigEndian, &cmd)
	return cmd, err
}

func PackBodyConfig(w io.Writer, connectAddr string) error {
	if err := binary.Write(w, binary.BigEndian, int16(len(connectAddr))); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, []byte(connectAddr)); err != nil {
		return err
	}
	return nil
}

func UnpackBodyConfig(r io.Reader) (connectAddr string, err error) {
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

func PackBodyTunID(w io.Writer, tunID int64) error {
	return binary.Write(w, binary.BigEndian, tunID)
}

func UnpackBodyTunID(r io.Reader) (tunID int64, err error) {
	err = binary.Read(r, binary.BigEndian, &tunID)
	return tunID, err
}

func PackBodyConnect(w io.Writer, connID int64) error {
	return binary.Write(w, binary.BigEndian, connID)
}

func UnpackBodyConnect(r io.Reader) (connID int64, err error) {
	err = binary.Read(r, binary.BigEndian, &connID)
	return connID, err
}

func PackBodyConnectResult(w io.Writer, connID int64, connectResult error) error {
	if err := binary.Write(w, binary.BigEndian, connID); err != nil {
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

func UnpackBodyConnectResult(r io.Reader) (connID int64, connectResult error, err error) {
	if err := binary.Read(r, binary.BigEndian, &connID); err != nil {
		return 0, connectResult, err
	}
	var n int16
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return connID, connectResult, err
	}
	if n > 0 {
		b := make([]byte, n)
		if err := binary.Read(r, binary.BigEndian, b); err != nil {
			return connID, connectResult, err
		}
		connectResult = errors.New(string(b))
	}
	return connID, connectResult, nil
}

func PackBodySend(w io.Writer, connID int64, data []byte) error {
	if err := binary.Write(w, binary.BigEndian, connID); err != nil {
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

func UnpackBodySend(r io.Reader) (connID int64, data []byte, err error) {
	if err := binary.Read(r, binary.BigEndian, &connID); err != nil {
		return connID, data, err
	}
	var n int64
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return connID, data, err
	}
	data = make([]byte, n)
	if err := binary.Read(r, binary.BigEndian, data); err != nil {
		return connID, data, err
	}
	return connID, data, nil
}

func PackBodyClose(w io.Writer, connID int64) error {
	return binary.Write(w, binary.BigEndian, connID)
}

func UnpackBodyClose(r io.Reader) (connID int64, err error) {
	err = binary.Read(r, binary.BigEndian, &connID)
	return connID, err
}
