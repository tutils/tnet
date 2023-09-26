package endpoint

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

var seq int16 = 0

func packHeader(w io.Writer, cmd Cmd) error {
	seq++
	err := binary.Write(w, binary.BigEndian, seq)
	if err != nil {
		return err
	}
	// fmt.Println("@@packHeader: seq", seq)
	err = binary.Write(w, binary.BigEndian, cmd)
	if err != nil {
		return err
	}
	return nil
}

func unpackHeader(r io.Reader) (seq int16, cmd Cmd, err error) {
	err = binary.Read(r, binary.BigEndian, &seq)
	// fmt.Println("@@unpackHeader: seq", seq)
	if err != nil {
		return seq, cmd, err
	}
	err = binary.Read(r, binary.BigEndian, &cmd)
	return seq, cmd, err
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

func packBodyTunID(w io.Writer, tunID int64) error {
	return binary.Write(w, binary.BigEndian, tunID)
}

func unpackBodyTunID(r io.Reader) (tunID int64, err error) {
	err = binary.Read(r, binary.BigEndian, &tunID)
	return tunID, err
}

func packBodyConnect(w io.Writer, connID int64) error {
	return binary.Write(w, binary.BigEndian, connID)
}

func unpackBodyConnect(r io.Reader) (connID int64, err error) {
	err = binary.Read(r, binary.BigEndian, &connID)
	return connID, err
}

func packBodyConnectResult(w io.Writer, connID int64, connectResult error) error {
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

func unpackBodyConnectResult(r io.Reader) (connID int64, connectResult error, err error) {
	err = binary.Read(r, binary.BigEndian, &connID)
	if err != nil {
		return connID, connectResult, err
	}
	var n int16
	if err = binary.Read(r, binary.BigEndian, &n); err != nil {
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

func packBodySend(w io.Writer, connID int64, data []byte) error {
	// fmt.Println("@@packBodySend: connID", connID, "dataLen", len(data))
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

func unpackBodySend(r io.Reader) (connID int64, data []byte, err error) {
	if err := binary.Read(r, binary.BigEndian, &connID); err != nil {
		return connID, data, err
	}
	var n int64
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return connID, data, err
	}
	// fmt.Println("@@unpackBodySend: connID", connID, "dataLen", n)
	data = make([]byte, n)
	if err := binary.Read(r, binary.BigEndian, data); err != nil {
		return connID, data, err
	}
	return connID, data, nil
}

func packBodyClose(w io.Writer, connID int64) error {
	return binary.Write(w, binary.BigEndian, connID)
}

func unpackBodyClose(r io.Reader) (connID int64, err error) {
	err = binary.Read(r, binary.BigEndian, &connID)
	return connID, err
}
