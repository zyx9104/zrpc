package main

import (
	"flag"

	"github.com/z-y-x233/zrpc"
)

var (
	addr         = flag.String("s", "127.0.0.1:5890", "addr")
	registryAddr = flag.String("h", "127.0.0.1:9999", "registryAddr")
	n            = flag.Int("n", 100, "loop times")
)

type S struct{}

type Args struct {
	X int
}

type Reply struct {
	X int
}

func main() {
	flag.Parse()
	c := zrpc.NewXClient(*registryAddr, "S", 0)
	reply := &Reply{}
	for i := 0; i < *n; i++ {
		c.Call("Test", Args{100}, reply)
		// log.Print(addr, reply.X)
	}
}
