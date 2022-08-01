package zrpc

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"reflect"
	"sync"

	"github.com/z-y-x233/zrpc/codec"
	"github.com/z-y-x233/zrpc/header"
	"github.com/z-y-x233/zrpc/option"
)

type Server struct {
	services sync.Map
}

type service struct {
	Name   string
	Method map[string]*methodType
}

type methodType struct {
	sync.Mutex // protects counters
	method     reflect.Method
	ArgType    reflect.Type
	ReplyType  reflect.Type
	numCalls   uint
}

func NewServer() *Server {
	return &Server{}
}

func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Print(err)
			return
		}
		server.ServeConn(conn)
	}
}

func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	opt := &option.Options{}
	js := json.NewDecoder(conn)
	err := js.Decode(opt)
	if err != nil {
		log.Print(err)
		return
	}
	Fcodec := codec.ServerCodecs[opt.CodecType]
	if Fcodec == nil {
		log.Print("ServeConn: not found CodecType ", opt.CodecType)
		return
	}
	go server.ServeCodec(Fcodec(conn))

}

func (server *Server) ServeCodec(codec codec.ServerCodec) {
	defer codec.Close()
	sending := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	for {
		req, body, err := server.ReadRequest(codec)
		if err != nil {
			log.Print(err)
			break
		}
		wg.Add(1)
		go server.call(codec, req, body, sending, wg)
	}
	wg.Wait()

}

func (server *Server) ReadRequest(codec codec.ServerCodec) (req *header.Request, body string, err error) {
	req, err = server.ReadRequestHeader(codec)
	if err != nil {
		log.Print("Server.ReadRequest ", err)
		return
	}
	var s string
	codec.ReadRequestBody(&s)
	return req, s, err
}

func (server *Server) ReadRequestHeader(codec codec.ServerCodec) (*header.Request, error) {
	req := header.GetRequest()
	err := codec.ReadRequestHeader(req)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (server *Server) SendResponse(resp *header.Response, reply string) {

}

func (server *Server) call(codec codec.ServerCodec, req *header.Request, body interface{}, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("rpc server read method: %v, seq: %v, args: %v", req.ServiceMethod, req.Seq, body)

	codec.WriteResponse(&header.Response{Seq: req.Seq, ServiceMethod: req.ServiceMethod}, &req.ServiceMethod)

	header.FreeRequest(req)
}
