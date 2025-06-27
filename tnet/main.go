package main

import (
	"log"
	"os"
	"strings"

	// _ "net/http/pprof"

	"github.com/tutils/tnet/cmd"
)

func mockCmdline(cmdline string) {
	os.Args = append(os.Args[:1], strings.Split(cmdline, " ")...)
}

func main() {
	// go http.ListenAndServe(":", nil)
	log.SetFlags(log.Ltime | log.Lshortfile)

	// mockCmdline(`httpsrv -l :28080`)

	cmd.Execute()
}
