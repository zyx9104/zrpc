package zrpc

import "sync"

type ClientCodec interface {
	ReadResponseHeader(resp *Response) error
	ReadResponseBody(body any) error
	WriteRequest(req *Request, body any) error
	Close() error
}

type Call struct {
}

type Client struct {
	codec ClientCodec

	reqMutex sync.Mutex // protects following
	request  Request

	mutex    sync.Mutex // protects following
	seq      uint64
	pending  map[uint64]*Call
	closing  bool // user has called Close
	shutdown bool // server has told us to stop
}
