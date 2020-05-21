package main

import (
	"github.com/tutils/tnet/cmd"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	go http.ListenAndServe(":", nil)
	log.SetFlags(log.Lshortfile)
	cmd.Execute()
}
