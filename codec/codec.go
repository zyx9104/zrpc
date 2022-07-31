package codec

import "github.com/z-y-x233/zrpc/header"

type ClientCodec interface {
	ReadResponseHead(response *header.Response) error
	ReadResponseBody(body interface{}) error
	WriteRequest(request *header.Request, body interface{}) error
	Close() error
}

type ServerCodec interface {
	ReadResponseHead(request *header.Request) error
	ReadRequestBody(body interface{}) error
	WriteResponse(response *header.Response, body interface{}) error
	Close() error
}
