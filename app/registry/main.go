package main

import (
	"flag"

	"github.com/z-y-x233/zrpc"
)

var (
	addr = flag.String("s", "127.0.0.1:9999", "addr")
)

func main() {
	flag.Parse()
	r := zrpc.NewRegistry()
	r.Listen(*addr)
}
