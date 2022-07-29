package zrpc

import (
	"log"
	"net"
	"net/rpc"

	"github.com/z-y-x233/zrpc/codec"
	"github.com/z-y-x233/zrpc/serialize"
)

type Server struct {
	*rpc.Server
	s serialize.Serializer
}

func NewServer(o *Options) *Server {
	return &Server{
		&rpc.Server{},
		serialize.Serializers[o.SerializeType],
	}
}

func (s *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Print("rpc.Serve: accept:", err.Error())
			return
		}
		go s.ServeCodec(codec.NewServerCodec(conn, s.s))
	}
}

func (s *Server) Register(rcvr interface{}) error {
	return s.Server.Register(rcvr)
}

// RegisterName is like Register but uses the provided name for the type
// instead of the receiver's concrete type.
func (s *Server) RegisterName(name string, rcvr interface{}) error {
	return s.Server.RegisterName(name, rcvr)
}

func (s *Server) ServeCodec(codec rpc.ServerCodec) {
	s.Server.ServeCodec(codec)
}

// ServeRequest is like ServeCodec but synchronously serves a single request.
// It does not close the codec upon completion.
func (s *Server) ServeRequest(codec rpc.ServerCodec) error {
	return s.Server.ServeRequest(codec)
}
