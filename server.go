package zrpc

import (
	"fmt"
	"go/token"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
)

var TypeOfError = reflect.TypeOf((*error)(nil)).Elem()

type any interface{}

type ServerCodec interface {
	ReadRequestHeader(req *Request) error
	ReadRequestBody(body any) error
	WriteResponse(resp *Response, body any) error
	Close() error
}

func defaultServerCodec(conn io.ReadWriteCloser) ServerCodec {
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
	return s.register(rcvr, "", false)
}

func (s *Server) RegisterName(rcvr any, name string) error {
	return s.register(rcvr, name, true)
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

func (s *Server) ServeCodec(codec ServerCodec) {
	wg := &sync.WaitGroup{}
	sending := &sync.Mutex{}
	for {
		svc, mtype, req, argv, replyv, err := s.ReadRequest(codec)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Print(err)
			if req != nil {
				s.sendResponse(codec, req, struct{}{}, err.Error(), sending)
				s.freeRequest(req)
			}
			continue
		}
		wg.Add(1)
		go svc.call(codec, s, mtype, req, argv, replyv, wg, sending)
	}

	wg.Wait()
	codec.Close()
}

func (s *Server) ReadRequest(codec ServerCodec) (svc *service, mtype *methodType, req *Request, argv, replyv reflect.Value, err error) {
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

func (s *Server) ReadRequestHeader(codec ServerCodec) (svc *service, mtype *methodType, req *Request, err error) {
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

func (s *service) call(codec ServerCodec, server *Server, mtype *methodType, req *Request, argv, replyv reflect.Value, wg *sync.WaitGroup, sending *sync.Mutex) {
	defer wg.Done()

	function := mtype.method.Func
	args := []reflect.Value{s.rcvr, argv, replyv}
	returnVal := function.Call(args)

	mtype.Lock()
	mtype.numCalls++
	mtype.Unlock()

	errInt := returnVal[0].Interface()
	errmsg := errInt.(error).Error()
	server.sendResponse(codec, req, replyv.Interface(), errmsg, sending)
	server.freeRequest(req)

}

func (s *Server) register(rcvr any, uname string, username bool) error {
	svc := new(service)
	svc.rcvr = reflect.ValueOf(rcvr)
	svc.typ = reflect.TypeOf(rcvr)
	sname := reflect.Indirect(svc.rcvr).Type().Name()

	if username {
		sname = uname
	}
	if !token.IsExported(sname) || sname == "" {
		err := fmt.Errorf("rpc register: service %q must have exported name", svc.typ)
		return err
	}
	svc.name = sname

	svc.method = getServiceMethods(svc.typ)

	if len(svc.method) == 0 {
		m := getServiceMethods(reflect.PointerTo(svc.typ))
		if len(m) != 0 {
			err := fmt.Errorf("rpc register: service %q should pass by pointer", svc.typ)
			log.Print(err)
			return err
		}
		err := fmt.Errorf("rpc register: service %q has no suitable method", svc.typ)
		log.Print(err)

		return err
	}
	log.Printf("rpc register service %q", svc.typ)
	return nil
}

func (s *Server) sendResponse(codec ServerCodec, req *Request, reply any, errmsg string, sending *sync.Mutex) {
	resp := s.getResponse()
	resp.Seq = req.Seq
	resp.ServiceMethod = req.ServiceMethod
	resp.Error = errmsg

	sending.Lock()
	err := codec.WriteResponse(resp, reply)
	sending.Unlock()

	if err != nil {
		log.Print("rpc.sendResponse: ", err)
	}
	s.freeResponse(resp)
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

func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return token.IsExported(t.Name()) || t.PkgPath() == ""
}

func getServiceMethods(svc reflect.Type) map[string]*methodType {
	methods := make(map[string]*methodType)

	for i := 0; i < svc.NumMethod(); i++ {
		method := svc.Method(i)
		mtype := method.Type
		if !method.IsExported() {
			continue
		}

		if mtype.NumIn() != 3 {
			continue
		}
		argType := mtype.In(1)

		if !isExportedOrBuiltinType(argType) {
			continue
		}

		replyType := mtype.In(2)

		if replyType.Kind() != reflect.Pointer {
			continue
		}

		if !isExportedOrBuiltinType(argType) {
			continue
		}

		if mtype.NumOut() != 1 {
			continue
		}

		if mtype.Out(0) != TypeOfError {
			continue
		}

		methods[method.Name] = &methodType{method: method, argType: argType, replyType: replyType}

	}

	return methods
}
