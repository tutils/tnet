package proxy

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"time"

	"github.com/tutils/tnet/endpoint/common"
	"golang.org/x/term"
)

func (h *proxyTunHandler) proxyPTY(ctx context.Context, tunID int64, tunr io.Reader, tunw io.Writer) int {
	opts := &h.p.opts

	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		log.Println("GetSize err", err)
		return 1
	}

	// send config: connect to pty
	buf := &bytes.Buffer{} // TODO: use pool
	if err := common.PackHeader(buf, common.CmdConnectPTY); err != nil {
		log.Println("packHeader err", err)
		return 1
	}

	if err := common.PackBodyConnectPTY(buf, opts.rawPTYMode, opts.executeArgs, int16(width), int16(height)); err != nil {
		log.Println("packBodyConnectPTY err", err)
		return 1
	}
	if _, err := tunw.Write(buf.Bytes()); err != nil {
		log.Println("write tun err", err)
		return 1
	}
	log.Printf("Write CmdConnectPTY, pty[%dx%d] %v", width, height, opts.executeArgs)

	fd := int(os.Stdin.Fd())
	var oldState *term.State

	// tun_reader -> conn_writer
	for {
		// select {
		// case <-errCh:
		// 	return
		// default:
		// }

		cmd, err := common.UnpackHeader(tunr)
		if err != nil {
			log.Println("unpackHeader err", err)
			return 1
		}
		switch cmd {
		case common.CmdConnectPTYResult:
			connectResult, err := common.UnpackBodyConnectPTYResult(tunr)
			if err != nil {
				log.Println("unpackBodyConnectPTYResult err", err)
				return 1
			}
			log.Printf("Read CmdConnectPTYResult, tunID %d, %v", tunID, connectResult)

			if !opts.rawPTYMode {
				oldState, err = term.MakeRaw(fd)
				if err != nil {
					log.Println("term err", err)
					return 1
				}
				defer term.Restore(fd, oldState)
			}

			go func() {
				ticker := time.NewTicker(time.Second)
				defer ticker.Stop()
				err := common.Copy(tunw, os.Stdin, func(tunw io.Writer, data []byte) error {
					select {
					case <-ticker.C:
						if w, h, err := term.GetSize(fd); err == nil && (w != width || h != height) {
							if err := common.PackHeader(tunw, common.CmdResizePTY); err != nil {
								return err
							}
							if err := common.PackBodyResizePTY(tunw, int16(w), int16(h)); err != nil {
								return err
							}
							width, height = w, h
						}
					default:
					}

					if err := common.PackHeader(tunw, common.CmdIOPTY); err != nil {
						return err
					}
					if err := common.PackBodyIOPTY(tunw, data); err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					// 异常终止
					log.Println("copy pty output err", err)
				}
			}()

		case common.CmdIOPTY:
			data, err := common.UnpackBodyIOPTY(tunr)
			if err != nil {
				log.Println("unpackBodyPTYIO err", err)
				return 1
			}
			if opts.rawPTYMode {
				if cr, ok := tunr.(*counterReader); ok {
					log.Printf("Read CmdIOPTY, tunID %d, %d bytes, download %s/s", tunID, len(data), humanReadable(uint64(cr.c.IncreaceRatePerSec())))
				} else {
					log.Printf("Read CmdIOPTY, tunID %d, %d bytes", tunID, len(data))
				}
			}
			if _, err := os.Stdout.Write(data); err != nil {
				log.Println("Write stdout err", err)
				return 1
			}

		case common.CmdClosePTY:
			exitCode, err := common.UnpackBodyClosePTY(tunr)
			if err != nil {
				log.Println("UnpackBodyClosePTY err", err)
				return 1
			}
			log.Printf("Read CmdClosePTY, tunID %d, exitCode %d", tunID, exitCode)
			return int(exitCode)
		}
	}
}
