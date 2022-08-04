package zrpc_test

import (
	"log"
	"net"
	"testing"

	"github.com/z-y-x233/zrpc"
)

type Svc struct{}

type Arg struct {
	X int
}

type Reply struct {
	X int
}

func (s *Svc) Test(arg Arg, reply *Reply) error {
	reply.X = arg.X * arg.X
	return nil
}

func init() {
	s := zrpc.NewServer()
	lis, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Fatal(err)
	}

	s.Register(new(Svc))
	err = s.Register(new(Svc))
	if err != nil {
		log.Print(err)
	}
	go s.Accept(lis)
}

func BenchmarkServer(b *testing.B) {
	conn, _ := net.Dial("tcp", ":9999")
	c := zrpc.NewClient(conn)
	re := &Reply{}
	for i := 0; i < b.N; i++ {
		c.Call("Svc.Test", Arg{i}, re)
		if re.X != i*i {
			log.Fatal(re)
		}
	}
}
