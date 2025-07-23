package common

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"

	"github.com/tutils/tnet/tun"
)

func SyncTunID(ctx context.Context, isServer bool, tunr io.Reader, tunw io.Writer) (int64, error) {
	if isServer {
		// send tunID
		tunID := ctx.Value(tun.ConnIDContextKey{}).(int64)
		buf := &bytes.Buffer{} // TODO: use pool
		if err := PackHeader(buf, CmdTunID); err != nil {
			log.Println("packHeader err", err)
			return 0, err
		}
		if err := PackBodyTunID(buf, tunID); err != nil {
			log.Println("packBodyTunID err", err)
			return 0, err
		}
		if _, err := tunw.Write(buf.Bytes()); err != nil {
			log.Println("write tun err", err)
			return 0, err
		}
		log.Printf("Write CmdTunID, TunID %d", tunID)
		return tunID, nil
	}

	// recv tunID
	cmd, err := UnpackHeader(tunr)
	if err != nil {
		log.Println("unpackHeader err", err)
		return 0, err
	}
	if cmd != CmdTunID {
		log.Println("invalid cmd")
		return 0, errors.New("wrong command, CmdTunID expected")
	}
	tunID, err := UnpackBodyTunID(tunr)
	if err != nil {
		log.Println("unpackBodyTunID err", err)
		return 0, err
	}
	log.Printf("Read CmdTunID, TunID %d", tunID)
	return tunID, nil
}
