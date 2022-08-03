package zrpc

import (
	"io"
	"log"
	"net"
	"sync"
)

type any interface{}

type ServeCodec interface {
	ReadRequestHeader(req *Request) error
	ReadRequestBody(body any) error
	WriteResponse(resp *Response, body any) error
	Close() error
}

type Request struct {
}

type Response struct {
}

type methodType struct {
}

type service struct {
}

type Server struct {
	serviceMap sync.Map

	reqLock sync.Mutex
	req     *Request

	respLock sync.Mutex
	resp     *Response
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Register(rcvr any) error {
	return nil
}

func (s *Server) RegisterName(rcvr any, name string) error {
	return nil
}

func (s *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Fatal("rpc listen: ", err)
		}
		go s.ServeConn(conn)
	}
}

func (s *Server) ServeConn(conn io.ReadWriteCloser) {

}
