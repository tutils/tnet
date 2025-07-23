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

	CmdConnectPTY
	CmdConnectPTYResult
	CmdResizePTY
	CmdIOPTY
	CmdClosePTY
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

func PackBodyConnectPTY(w io.Writer, rawMode bool, args []string, width int16, height int16) error {
	var mode uint8
	if rawMode {
		mode = 1
	}
	if err := binary.Write(w, binary.BigEndian, mode); err != nil {
		return err
	}
	n := int16(len(args))
	if err := binary.Write(w, binary.BigEndian, n); err != nil {
		return err
	}
	for _, arg := range args {
		if err := binary.Write(w, binary.BigEndian, int16(len(arg))); err != nil {
			return err
		}
		if _, err := w.Write([]byte(arg)); err != nil {
			return err
		}
	}
	if err := PackBodyResizePTY(w, width, height); err != nil {
		return err
	}
	return nil
}

func UnpackBodyConnectPTY(r io.Reader) (rawMode bool, args []string, width int16, height int16, err error) {
	var mode uint8
	if err := binary.Read(r, binary.BigEndian, &mode); err != nil {
		return false, nil, 0, 0, err
	}
	if mode == 1 {
		rawMode = true
	}

	var n int16
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return false, nil, 0, 0, err
	}

	for i := int16(0); i < n; i++ {
		var argLen int16
		if err := binary.Read(r, binary.BigEndian, &argLen); err != nil {
			return false, nil, 0, 0, err
		}
		b := make([]byte, argLen)
		if _, err := io.ReadAtLeast(r, b, int(argLen)); err != nil {
			return false, nil, 0, 0, err
		}
		args = append(args, string(b))
	}

	if width, height, err = UnpackBodyResizePTY(r); err != nil {
		return false, nil, 0, 0, err
	}

	return rawMode, args, width, height, nil
}

func PackBodyConnectPTYResult(w io.Writer, connectResult error) error {
	// if err := binary.Write(w, binary.BigEndian, int16(len(output))); err != nil {
	// 	return err
	// }
	// if _, err := w.Write([]byte(output)); err != nil {
	// 	return err
	// }
	if connectResult != nil {
		if err := binary.Write(w, binary.BigEndian, int16(len(connectResult.Error()))); err != nil {
			return err
		}
		if _, err := w.Write([]byte(connectResult.Error())); err != nil {
			return err
		}
	} else {
		if err := binary.Write(w, binary.BigEndian, int16(0)); err != nil {
			return err
		}
	}
	return nil
}

func UnpackBodyConnectPTYResult(r io.Reader) (connectResult error, err error) {
	var n int16
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return nil, err
	}
	if n > 0 {
		b := make([]byte, n)
		if err := binary.Read(r, binary.BigEndian, b); err != nil {
			return nil, err
		}
		connectResult = errors.New(string(b))
	}
	return connectResult, nil
}

func PackBodyResizePTY(w io.Writer, width int16, height int16) error {
	if err := binary.Write(w, binary.BigEndian, width); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, height); err != nil {
		return err
	}
	return nil
}

func UnpackBodyResizePTY(r io.Reader) (width int16, height int16, err error) {
	if err := binary.Read(r, binary.BigEndian, &width); err != nil {
		return 0, 0, err
	}
	if err := binary.Read(r, binary.BigEndian, &height); err != nil {
		return 0, 0, err
	}
	return width, height, nil
}

func PackBodyIOPTY(w io.Writer, data []byte) error {
	n := int32(len(data))
	if err := binary.Write(w, binary.BigEndian, n); err != nil {
		return err
	}
	nw, err := w.Write(data)
	if err != nil {
		return err
	}
	if nw < 0 || nw > len(data) {
		return errors.New("invalid write result")
	}
	if nw < len(data) {
		return io.ErrShortWrite
	}
	return nil
}

func UnpackBodyIOPTY(r io.Reader) (data []byte, err error) {
	var n int32
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return nil, err
	}
	data = make([]byte, n)
	if _, err := io.ReadAtLeast(r, data, int(n)); err != nil {
		return nil, err
	}
	return data, nil
}

func PackBodyClosePTY(w io.Writer, exitCode int64) error {
	return binary.Write(w, binary.BigEndian, exitCode)
}

func UnpackBodyClosePTY(r io.Reader) (exitCode int64, err error) {
	err = binary.Read(r, binary.BigEndian, &exitCode)
	return exitCode, err
}
