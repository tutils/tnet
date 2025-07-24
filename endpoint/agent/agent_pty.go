package agent

import (
	"context"
	"io"
	"log"

	"github.com/aymanbagabas/go-pty"
	"github.com/tutils/tnet/endpoint/common"
	"golang.org/x/term"
)

func (h *agentTunHandler) agentPTY(ctx context.Context, tunID int64, tunr io.Reader, tunw io.Writer) {
	rawMode, executeArgs, width, height, err := common.UnpackBodyConnectPTY(tunr)
	if err != nil {
		log.Println("unpackBodyConnectPTY err", err)
		return
	}
	log.Printf("Read CmdConnectPTY, args %v, size %dx%d, raw %v", executeArgs, width, height, rawMode)

	writeConnectResult := func(err error) error {
		if err := common.PackHeader(tunw, common.CmdConnectPTYResult); err != nil {
			log.Println("packHeader err", err)
			return err
		}
		if err := common.PackBodyConnectPTYResult(tunw, err); err != nil {
			log.Println("packBodyConnectPTYResult err", err)
			return err
		}
		log.Println("Write CmdConnectPTYResult")
		return nil
	}

	// connected := false
	p, err := pty.New()
	if err != nil {
		log.Println("new pty err", err)
		writeConnectResult(err)
		return
	}
	defer p.Close()

	if rawMode {
		term.MakeRaw(int(p.Fd()))
	}

	if err := p.Resize(int(width), int(height)); err != nil {
		log.Println("resize pty err", err)
	}

	cmd := p.CommandContext(ctx, executeArgs[0], executeArgs[1:]...)
	err = cmd.Start()
	if err := writeConnectResult(err); err != nil {
		return
	}
	if err != nil {
		log.Println("start pty cmd err", err)
		return
	}

	// connected = true

	// var wg sync.WaitGroup
	// wg.Add(1)
	exitCodeCh := make(chan int)
	go func() {
		cmd.Wait()
		log.Printf("pty cmd exited")
		// 子进程退出，终止pty输出拷贝循环
		p.Close()
		exitCodeCh <- cmd.ProcessState.ExitCode()
	}()

	exitCh := make(chan struct{})
	go func() {
		defer close(exitCh)

		if err := common.Copy(tunw, p, func(tunw io.Writer, data []byte) error {
			if err := common.PackHeader(tunw, common.CmdIOPTY); err != nil {
				return err
			}
			if err := common.PackBodyIOPTY(tunw, data); err != nil {
				return err
			}
			return nil
		}); err != nil {
			// 异常终止
			log.Println("copy pty output err", err)
		}
		log.Printf("copy pty output loop exited")

		// pty输出拷贝循环退出，需要终止子进程
		cmd.Process.Kill()
		exitCode := <-exitCodeCh
		if err := common.PackHeader(tunw, common.CmdClosePTY); err != nil {
			log.Println("packHeader err", err)
			return
		}
		if err := common.PackBodyClosePTY(tunw, int64(exitCode)); err != nil {
			log.Println("packHeader err", err)
			return
		}
	}()

LOOP:
	for {
		cmd, err := common.UnpackHeader(tunr)
		if err != nil {
			log.Println("unpackHeader err", err)
			break
		}
		switch cmd {
		case common.CmdResizePTY:
			width, height, err := common.UnpackBodyResizePTY(tunr)
			if err != nil {
				log.Println("unpackBodyResizePTY err", err)
				break LOOP
			}
			log.Printf("Read CmdResizePTY, tunID %d, size %dx%d", tunID, width, height)
			if err := p.Resize(int(width), int(height)); err != nil {
				log.Println("resize pty err", err)
			}

		case common.CmdIOPTY:
			data, err := common.UnpackBodyIOPTY(tunr)
			if err != nil {
				log.Println("unpackBodyIOPTY err", err)
				break LOOP
			}
			log.Printf("Read CmdIOPTY, tunID %d, %d bytes", tunID, len(data))
			if _, err := p.Write(data); err != nil {
				log.Println("write pty err", err)
				break LOOP
			}
		}
	}
	cmd.Process.Kill()
	<-exitCh
}
