package codec

import (
	"bufio"
	"encoding/gob"
	"github.com/z-y-x233/zrpc/header"
	"io"
	"log"
)

func NewGobClientCodec(conn io.ReadWriteCloser) ClientCodec {
	buf := bufio.NewWriter(conn)
	return &GobClientCodec{
		conn: conn,
		enc:  gob.NewEncoder(buf),
		dec:  gob.NewDecoder(conn),
		buf:  buf,
	}
}

type GobClientCodec struct {
	conn   io.ReadWriteCloser
	enc    *gob.Encoder
	dec    *gob.Decoder
	buf    *bufio.Writer
	closed bool
}

func (cc *GobClientCodec) ReadResponseHead(response *header.Response) error {
	return cc.dec.Decode(response)
}

func (cc *GobClientCodec) ReadResponseBody(body interface{}) error {
	return cc.dec.Decode(body)
}

func (cc *GobClientCodec) WriteRequest(request *header.Request, body interface{}) error {
	if err := cc.enc.Encode(request); err != nil {
		log.Print("rpc WriteRequest header err: ", err)

		return err
	}
	if err := cc.enc.Encode(body); err != nil {
		log.Print("rpc WriteRequest body err: ", err)
		return err
	}
	return cc.buf.Flush()
}

func (cc *GobClientCodec) Close() error {
	if cc.closed {
		return nil
	}
	cc.closed = true
	return cc.conn.Close()
}
