package zrpc

import (
	"log"
	"net"
	"testing"

	"github.com/z-y-x233/zrpc/compress"
	"github.com/z-y-x233/zrpc/serialize"
)

var s *Server

type S struct{}

type Args struct {
	A int
}

type Reply struct {
	A int
}

func (*S) Increment(args Args, reply *Reply) error {
	reply.A = args.A + 1
	return nil
}

func init() {
	lis, _ := net.Listen("tcp", ":9999")

	s = NewServer(&Options{serialize.Json, 0})
	s.Register(new(S))
	go s.Accept(lis)
}

func TestClient(t *testing.T) {
	conn, _ := net.Dial("tcp", ":9999")
	c := NewClient(conn, &Options{serialize.Json, compress.Raw})
	args := Args{1}
	reply := &Reply{}
	err := c.Call("S.Increment", args, reply)

	if err != nil {
		log.Fatal(err)
	}
	log.Print(reply.A)

}
