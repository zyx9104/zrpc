package zrpc

import (
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
)

type any interface{}

type ServeCodec interface {
	ReadRequestHeader(req *Request) error
	ReadRequestBody(body any) error
	WriteResponse(resp *Response, body any) error
	Close() error
}

func defaultServerCodec(conn io.ReadWriteCloser) ServeCodec {
	return nil
}

type Request struct {
	Seq           uint64
	ServiceMethod string
}

type Response struct {
	Seq           uint64
	ServiceMethod string
	Error         string
}

type methodType struct {
	sync.Mutex
	method    reflect.Method
	argType   reflect.Type
	replyType reflect.Type
	numCalls  uint
}

type service struct {
	name   string
	rcvr   reflect.Value
	typ    reflect.Type
	method map[string]*methodType
}

type Server struct {
	serviceMap sync.Map
	reqPool    sync.Pool
	respPool   sync.Pool
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
	codec := defaultServerCodec(conn)

	s.ServeCodec(codec)
}

func (s *Server) ServeCodec(codec ServeCodec) {

	for {
		svc, mtype, req, argv, replyv, err := s.ReadRequest(codec)
		if err != nil {
			if err != io.EOF {
				log.Print(err)
			}
			continue
		}

		go svc.call(s, mtype, req, argv, replyv)
	}

}

func (s *Server) ReadRequest(codec ServeCodec) (svc *service, mtype *methodType, req *Request, argv, replyv reflect.Value, err error) {
	svc, mtype, req, err = s.ReadRequestHeader(codec)
	if err != nil {
		codec.ReadRequestBody(nil)
		return
	}

	isV := false
	if mtype.argType.Kind() == reflect.Pointer {
		argv = reflect.New(mtype.argType.Elem())
	} else {
		isV = true
		argv = reflect.New(mtype.argType)
	}

	if err = codec.ReadRequestBody(argv.Interface()); err != nil {
		return
	}

	if isV {
		argv = argv.Elem()
	}

	replyv = reflect.New(mtype.replyType.Elem())

	switch mtype.replyType.Elem().Kind() {
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(mtype.replyType.Elem(), 0, 0))
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(mtype.replyType.Elem()))
	}
	return
}

func (s *Server) ReadRequestHeader(codec ServeCodec) (svc *service, mtype *methodType, req *Request, err error) {
	req = s.getRequest()
	err = codec.ReadRequestHeader(req)
	if err != nil {
		return
	}

	dot := strings.LastIndex(req.ServiceMethod, ".")
	serviceName := req.ServiceMethod[:dot]
	methodName := req.ServiceMethod[dot+1:]

	svic, ok := s.serviceMap.Load(serviceName)
	if !ok {
		err = fmt.Errorf("rpc service %q not exist", serviceName)
		return
	}
	svc = svic.(*service)

	mtype, ok = svc.method[methodName]
	if !ok {
		err = fmt.Errorf("rpc service %q not have method %q", svc.typ, methodName)
		return
	}

	return
}

//===============================

func (s *service) call(service *Server, mtype *methodType, req *Request, argv, replyv reflect.Value) {
	// function := mtype.method.Func

	// args := []reflect.Value{s.rcvr, argv, replyv}
	// returnVal := function.Call(args)

}

func (s *Server) getRequest() *Request {
	return s.reqPool.Get().(*Request)
}

func (s *Server) freeRequest(req *Request) {
	req.reset()
	s.reqPool.Put(req)
}

func (s *Server) getResponse() *Response {
	return s.respPool.Get().(*Response)
}

func (s *Server) freeResponse(resp *Response) {
	resp.reset()
	s.respPool.Put(resp)
}

// ===========================

func (req *Request) reset() {
	req.ServiceMethod = ""
	req.Seq = 0
}

func (resp *Response) reset() {
	resp.Seq = 0
	resp.ServiceMethod = ""
	resp.Error = ""
}
