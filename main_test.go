package zrpc

import (
	"log"
	"net"
	"testing"
)

type A struct{}

type aa struct{}

type Args struct {
	X int
}

type Reply struct {
	X int
}

func (*A) Inc(args Args, reply *Reply) error {
	reply.X = args.X + 114514
	return nil
}

func (*A) Test(args *Args, reply *Reply) error {
	return nil
}

func TestRpc(t *testing.T) {
	s := NewServer()
	lis, _ := net.Listen("tcp", ":4321")
	s.Register(new(A))
	go s.Accept(lis)

	c := NewClient()
	c.Dial("tcp", ":4321")
	reply := &Reply{}
	c.Call("A.Inc", &Args{1}, reply)
	log.Print(reply.X)
}
