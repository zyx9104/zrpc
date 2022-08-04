package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"github.com/z-y-x233/zrpc"
)

var (
	addr         = flag.String("s", "127.0.0.1:6890", "addr")
	registryAddr = flag.String("h", "127.0.0.1:9999", "registryAddr")
)

type S struct{}

type Args struct {
	X int
}

type Reply struct {
	X int
}

func (*S) Test(args Args, reply *Reply) error {
	reply.X = args.X * args.X
	return nil
}

func main() {
	flag.Parse()
	s := zrpc.NewServer()
	s.Register(new(S))
	resp, err := http.Get(fmt.Sprintf("http://%s/register?serviceName=%s&addr=%s", *registryAddr, "S", *addr))
	if err != nil {
		log.Fatal(err)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	log.Print(string(body))
	lis, _ := net.Listen("tcp", *addr)
	for {
		conn, err := lis.Accept()
		log.Println(*addr, "accept one connect")
		if err != nil {
			log.Fatal("rpc listen: ", err)
		}
		go s.ServeConn(conn)
	}
}
