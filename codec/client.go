package codec

import (
	"bufio"
	"io"
	"net/rpc"

	"github.com/z-y-x233/zrpc/header"
	"github.com/z-y-x233/zrpc/serialize"
)

type ClientCodec struct {
	r    *bufio.Reader
	w    *bufio.Writer
	c    io.ReadWriteCloser
	s    serialize.Serializer
	resp *header.ResponseHeader
}

func (cc *ClientCodec) ReadResponseHeader(resp *rpc.Response) error {
	// err := cc.req.Read(cc.r)
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (cc *ClientCodec) ReadResponseBody(body interface{}) error {
	return nil

}

func (cc *ClientCodec) WriteRequest(req *rpc.Request, body interface{}) error {
	return nil

}

func (cc *ClientCodec) Close() error {
	return cc.c.Close()

}
