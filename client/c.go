package main

import (
	"log"
	"net"

	"github.com/z-y-x233/zrpc"
)

type Svc struct{}

type Arg struct {
	X int
}

type Reply struct {
	X int
}

func main() {
	conn, _ := net.Dial("tcp", ":9999")
	c := zrpc.NewClient(conn)
	re := &Reply{}
	c.Call("Svc.Test", Arg{11}, re)
	log.Print(re.X)
}
