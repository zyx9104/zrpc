package codec

import (
	"io"

	"github.com/z-y-x233/zrpc/header"
)

type Type uint8

type ClientCodec interface {
	ReadResponseHeader(response *header.Response) error
	ReadResponseBody(body interface{}) error
	WriteRequest(request *header.Request, body interface{}) error
	Close() error
}

type ServerCodec interface {
	ReadRequestHeader(request *header.Request) error
	ReadRequestBody(body interface{}) error
	WriteResponse(response *header.Response, body interface{}) error
	Close() error
}

const (
	Invalid Type = iota
	Gob
)

var ServerCodecs = map[Type]func(io.ReadWriteCloser) ServerCodec{
	Invalid: nil,
	Gob:     NewGobServerCodec,
}

var ClientCodecs = map[Type]func(io.ReadWriteCloser) ClientCodec{
	Invalid: nil,
	Gob:     NewGobClientCodec,
}
