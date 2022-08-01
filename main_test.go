package zrpc

import (
	"net"
	"testing"
	"time"
)

func TestRpc(t *testing.T) {
	s := NewServer()
	lis, _ := net.Listen("tcp", ":4321")

	go s.Accept(lis)

	c := NewClient()
	body := "123"
	c.Dial("tcp", ":4321")
	c.Call("Test.Test", "666", &body)

	d := make(chan *Call, 10)
	for i := 0; i < 5; i++ {
		c.Go("Test.Test", "123", &body, d)
	}
	time.Sleep(time.Second)
}
