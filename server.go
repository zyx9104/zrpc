package zrpc

import (
	"encoding/json"
	"errors"
	"go/token"
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
			continue
		}

		argv := mtype.In(1)
		//Argv 必须是导出的
		if isExportedOrBuiltIn(argv) {
			continue
		}

		reply := mtype.In(2)
		//reply 必须为指针
		if reply.Kind() != reflect.Pointer {
			continue
		}
		//reply 必须是导出的
		if isExportedOrBuiltIn(reply) {
			continue
		}
		//返回值数量为1
		if mtype.NumOut() != 1 {
			continue
		}

		//返回值类型必须为error
		if mtype.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
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
