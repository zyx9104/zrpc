package zrpc

import (
	"log"
	"net"
	"testing"
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

func TestRpc(t *testing.T) {

	s := NewServer()
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

	conn, _ := net.Dial("tcp", ":9999")
	c := NewClient(conn)
	re := &Reply{}
	c.Call("Svc.Test", Arg{11}, re)
	log.Print(re.X)

	// time.Sleep(time.Second)
}

func BenchmarkServer(b *testing.B) {
	for i := 0; i < b.N; i++ {

	}
}
