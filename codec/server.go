package codec

import (
	"bufio"
	"fmt"
	"io"
	"net/rpc"
	"sync"

	"github.com/z-y-x233/zrpc/compress"
	"github.com/z-y-x233/zrpc/header"
	"github.com/z-y-x233/zrpc/serialize"
)

type ctx struct {
	CompressType compress.Type
}

type ServerCodec struct {
	r       *bufio.Reader
	w       *bufio.Writer
	c       io.ReadWriteCloser
	s       serialize.Serializer
	req     *header.RequestHeader
	closed  bool
	ctxLock sync.Mutex
	pending map[uint64]*ctx
}

func NewServerCodec(conn io.ReadWriteCloser, s serialize.Serializer) *ServerCodec {
	return &ServerCodec{
		r:       bufio.NewReader(conn),
		w:       bufio.NewWriter(conn),
		c:       conn,
		s:       s,
		req:     header.GetRequest(),
		ctxLock: sync.Mutex{},
		pending: make(map[uint64]*ctx),
	}
}

func (cc *ServerCodec) ReadRequestHeader(req *rpc.Request) error {

	err := cc.req.Read(cc.r)
	if err != nil {
		return err
	}

	cc.ctxLock.Lock()
	cc.pending[req.Seq] = &ctx{CompressType: cc.req.CompressType}
	cc.ctxLock.Unlock()

	req.Seq = cc.req.Seq
	req.ServiceMethod = cc.req.ServiceMethod

	return nil
}

func (cc *ServerCodec) ReadRequestBody(body interface{}) error {

	if body == nil {
		if cc.req.RequestLen != 0 {
			if _, err := cc.r.Read(make([]byte, cc.req.RequestLen)); err != nil {
				return err
			}
		}
		return nil
	}

	data := make([]byte, cc.req.RequestLen)
	n, err := cc.r.Read(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("ServerCodec.ReadRequestBody error: read %v bytes, should %v byte", n, len(data))
	}
	compressor := compress.Compressors[cc.req.CompressType]
	unzipData, err := compressor.Unzip(data)
	if err != nil {
		return nil
	}

	return cc.s.Unmarshal(unzipData, body)
}

func (cc *ServerCodec) WriteResponse(resp *rpc.Response, body interface{}) error {

	cc.ctxLock.Lock()
	ctx, ok := cc.pending[resp.Seq]
	if !ok {
		cc.ctxLock.Unlock()
		return fmt.Errorf("ServerCodec.WriteResponse error: not found req: %v, method: %v", resp.Seq, resp.ServiceMethod)
	}
	delete(cc.pending, resp.Seq)
	cc.ctxLock.Unlock()

	head := header.GetResponse()
	defer header.FreeResponse(head)

	head.Seq = resp.Seq
	head.Error = resp.Error
	head.ErrorLen = uint16(len(head.Error))
	head.CompressType = ctx.CompressType

	compressor, ok := compress.Compressors[head.CompressType]
	if !ok {
		return fmt.Errorf("ServerCodec.WriteResponse error: not found compressType: %v", head.CompressType)
	}
	data, err := cc.s.Marshal(body)
	if err != nil {
		return fmt.Errorf("ServerCodec.WriteResponse: %v", err)
	}

	zipData, err := compressor.Zip(data)

	if err != nil {
		return fmt.Errorf("ServerCodec.WriteResponse: %v", err)
	}

	head.ResponseLen = uint32(len(zipData))

	head.Write(cc.w)
	cc.w.Write(zipData)

	return cc.w.Flush()
}

func (cc *ServerCodec) Close() error {
	if cc.closed {
		return nil
	}
	cc.closed = true
	return cc.c.Close()
}
