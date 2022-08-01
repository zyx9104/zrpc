package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"

	"github.com/z-y-x233/zrpc/header"
)

type GobServerCodec struct {
	conn   io.ReadWriteCloser
	enc    *gob.Encoder
	dec    *gob.Decoder
	buf    *bufio.Writer
	closed bool
}

func NewGobServerCodec(conn io.ReadWriteCloser) ServerCodec {
	buf := bufio.NewWriter(conn)
	return &GobServerCodec{
		conn: conn,
		enc:  gob.NewEncoder(conn),
		dec:  gob.NewDecoder(conn),
		buf:  buf,
	}
}

func (cc *GobServerCodec) ReadRequestHeader(request *header.Request) error {
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
