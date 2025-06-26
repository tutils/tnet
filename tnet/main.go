package main

import (
	"log"

	// _ "net/http/pprof"

	"github.com/tutils/tnet/cmd"
)

func main() {
	// go http.ListenAndServe(":", nil)
	log.SetFlags(log.Ltime | log.Lshortfile)
	cmd.Execute()
}
