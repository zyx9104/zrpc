package zrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/token"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"

	"github.com/z-y-x233/zrpc/codec"
	"github.com/z-y-x233/zrpc/header"
	"github.com/z-y-x233/zrpc/option"
)

type Server struct {
	serviceMap sync.Map
}

type service struct {
	name   string
	typ    reflect.Type
	rcv    reflect.Value
	method map[string]*methodType
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
		s, mtype, req, args, reply, err := server.ReadRequest(codec)
		if err != nil {
			log.Print(err)
			server.SendResponse(codec, req, struct{}{}, sending, err.Error())
			header.FreeRequest(req)
			break
		}
		wg.Add(1)
		go s.call(server, sending, wg, mtype, req, args, reply, codec)
	}
	wg.Wait()

}

func (server *Server) ReadRequest(codec codec.ServerCodec) (s *service, mtype *methodType, req *header.Request, args, reply reflect.Value, err error) {
	s, mtype, req, err = server.ReadRequestHeader(codec)

	if err != nil {
		codec.ReadRequestBody(nil)
		return
	}

	if mtype.ArgType.Kind() == reflect.Pointer {
		args = reflect.New(mtype.ArgType.Elem())
	} else {
		args = reflect.New(mtype.ArgType)
	}

	if err = codec.ReadRequestBody(args.Interface()); err != nil {
		return
	}

	if mtype.ArgType.Kind() != reflect.Pointer {
		args = args.Elem()
	}

	reply = reflect.New(mtype.ReplyType.Elem())
	switch mtype.ReplyType.Elem().Kind() {
	case reflect.Map:
		reply.Elem().Set(reflect.MakeMap(mtype.ReplyType.Elem()))
	case reflect.Slice:
		reply.Elem().Set(reflect.MakeSlice(mtype.ReplyType.Elem(), 0, 0))
	}
	return
}

func (server *Server) ReadRequestHeader(codec codec.ServerCodec) (s *service, mtype *methodType, req *header.Request, err error) {
	req = header.GetRequest()
	err = codec.ReadRequestHeader(req)
	if err != nil {
		return
	}
	dot := strings.LastIndex(req.ServiceMethod, ".")
	serviceName := req.ServiceMethod[:dot]
	methodName := req.ServiceMethod[dot+1:]

	svic, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = fmt.Errorf("not found service %s", serviceName)
		return
	}
	s = svic.(*service)
	mtype, ok = s.method[methodName]
	if !ok {
		err = fmt.Errorf("service %s not found method %s", serviceName, methodName)
		return
	}
	return
}

func (server *Server) SendResponse(codec codec.ServerCodec, req *header.Request, reply interface{}, sending *sync.Mutex, errmsg string) error {
	resp := header.GetResponse()
	resp.ServiceMethod = req.ServiceMethod
	resp.Seq = req.Seq
	resp.Error = errmsg

	sending.Lock()
	defer sending.Unlock()
	return codec.WriteResponse(resp, reply)
}

func (s *service) call(server *Server, sending *sync.Mutex, wg *sync.WaitGroup, mtype *methodType, req *header.Request, args, reply reflect.Value, codec codec.ServerCodec) {
	if wg != nil {
		defer wg.Done()
	}
	mtype.Lock()
	mtype.numCalls++
	mtype.Unlock()
	fun := mtype.method.Func

	Param := []reflect.Value{s.rcv, args, reply}
	returnVal := fun.Call(Param)

	errInter := returnVal[0].Interface()
	errmsg := ""
	if errInter != nil {
		errmsg = errInter.(error).Error()
	}
	server.SendResponse(codec, req, reply.Interface(), sending, errmsg)
	header.FreeRequest(req)
}

/*
服务注册
*/

func (server *Server) Register(rcvr interface{}) {
	server.register(rcvr, "", false)
}

func (server *Server) RegisterName(rcvr interface{}, name string) {
	server.register(rcvr, name, true)
}

func (server *Server) register(rcvr interface{}, name string, username bool) error {
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcv = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.rcv).Type().Name()
	if username {
		sname = name
	}

	if sname == "" {
		s := "zrpc.Register: no service name for type " + s.typ.String()
		log.Print(s)
		return errors.New(s)
	}
	if !token.IsExported(sname) && !username {
		s := "zrpc.Register: service " + sname + " must be exported"
		log.Print(s)
		return errors.New(s)
	}
	s.name = sname
	s.method = getMethods(s.typ)
	if len(s.method) == 0 {
		str := ""
		method := getMethods(reflect.PointerTo(s.typ))
		if len(method) != 0 {
			str = "zrpc.Register: type " + sname + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			str = "zrpc.Register: type " + sname + " has no exported methods of suitable type"
		}
		log.Print(str)
		return errors.New(str)
	}
	if _, dup := server.serviceMap.LoadOrStore(sname, s); dup {
		return errors.New("zrpc: service already defined: " + sname)
	}
	return nil
}

func isExportedOrBuiltIn(typ reflect.Type) bool {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return token.IsExported(typ.Name()) || typ.PkgPath() == ""

}

func getMethods(svc reflect.Type) map[string]*methodType {
	methods := make(map[string]*methodType)
	for i := 0; i < svc.NumMethod(); i++ {
		method := svc.Method(i)
		//方法必须是导出的
		if !method.IsExported() {
			continue
		}
		mtype := method.Type
		mname := method.Name
		//必须为三个参数(rcvr)f(argv,reply)
		if mtype.NumIn() != 3 {
			log.Printf("zrpc.Register: method %q has %d input parameters; needs exactly three\n", mname, mtype.NumIn())

			continue
		}

		argv := mtype.In(1)
		//Argv 必须是导出的
		if !isExportedOrBuiltIn(argv) {
			log.Printf("zrpc.Register: method %q, req type %q is not exported\n", mname, argv)

			continue
		}

		reply := mtype.In(2)
		//reply 必须为指针
		if reply.Kind() != reflect.Pointer {
			log.Printf("zrpc.Register: method %q reply type %q must be pointer\n", mname, reply)

			continue
		}
		//reply 必须是导出的
		if !isExportedOrBuiltIn(reply) {
			log.Printf("zrpc.Register: method %q, reply type %q is not exported\n", mname, reply)

			continue
		}
		//返回值数量为1
		if mtype.NumOut() != 1 {
			log.Printf("zrpc.Register: method %q has %d output parameters; needs exactly one\n", mname, mtype.NumOut())
			continue
		}

		//返回值类型必须为error
		if mtype.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			log.Printf("rpc.Register: return type of method %q is %q, must be error\n", mname, mtype.Out(0))
			continue
		}

		methods[mname] = &methodType{
			method:    method,
			ArgType:   argv,
			ReplyType: reply,
		}
	}
	return methods
}
