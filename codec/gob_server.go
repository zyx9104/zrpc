package codec

import (
	"bufio"
	"encoding/gob"
	"github.com/z-y-x233/zrpc/header"
	"io"
	"log"
)

type GobServerCodec struct {
	conn   io.ReadWriteCloser
	enc    *gob.Encoder
	dec    *gob.Decoder
	buf    *bufio.Writer
	closed bool
}

func (cc *GobServerCodec) ReadResponseHead(request *header.Request) error {
	return cc.dec.Decode(request)
}

func (cc *GobServerCodec) ReadRequestBody(body interface{}) error {
	return cc.dec.Decode(body)
}

func (cc *GobServerCodec) WriteResponse(response *header.Response, body interface{}) error {
	if err := cc.enc.Encode(response); err != nil {
		log.Print("rpc.WriteResponse header err: ", err)

		return err
	}
	if err := cc.enc.Encode(body); err != nil {
		log.Print("rpc.WriteResponse body err: ", err)
		return err
	}
	return cc.buf.Flush()
}

func (cc *GobServerCodec) Close() error {
	if cc.closed {
		return nil
	}
	cc.closed = true
	return cc.conn.Close()
}
