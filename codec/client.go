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

type ClientCodec struct {
	r            *bufio.Reader
	w            *bufio.Writer
	c            io.ReadWriteCloser
	s            serialize.Serializer
	resp         *header.ResponseHeader
	CompressType compress.Type

	mtx     sync.Mutex
	pending map[uint64]string
}

func NewClientCodec(conn io.ReadWriteCloser, s serialize.Serializer) *ClientCodec {

	return &ClientCodec{
		r:       bufio.NewReader(conn),
		w:       bufio.NewWriter(conn),
		c:       conn,
		s:       s,
		resp:    header.GetResponse(),
		mtx:     sync.Mutex{},
		pending: make(map[uint64]string),
	}
}

func (cc *ClientCodec) ReadResponseHeader(resp *rpc.Response) error {

	err := cc.resp.Read(cc.r)
	if err != nil {
		return err
	}
	cc.mtx.Lock()
	method, ok := cc.pending[cc.resp.Seq]
	if !ok {
		cc.mtx.Unlock()
		return fmt.Errorf("ClientCodec.ReadResponseHeader not found method seq %v", cc.resp.Seq)
	}
	delete(cc.pending, cc.resp.Seq)
	cc.mtx.Unlock()

	resp.Seq = cc.resp.Seq
	resp.Error = cc.resp.Error
	resp.ServiceMethod = method

	return nil
}

func (cc *ClientCodec) ReadResponseBody(body interface{}) error {

	if body == nil {
		if cc.resp.ResponseLen != 0 {
			if _, err := cc.r.Read(make([]byte, cc.resp.ResponseLen)); err != nil {
				return err
			}
		}
		return nil
	}

	data := make([]byte, cc.resp.ResponseLen)

	compressor, ok := compress.Compressors[cc.CompressType]
	if !ok {
		return fmt.Errorf("ClientCodec.WriteRequest error: not found compressType: %v", cc.CompressType)
	}

	n, err := cc.r.Read(data)
	if err != nil {
		return fmt.Errorf("ClientCodec.ReadResponseBody read error: %v", err)
	}
	if n != len(data) {
		return fmt.Errorf("ClientCodec.ReadResponseBody read error: read %v bytes, should %v byte", n, len(data))

	}

	unzipData, err := compressor.Unzip(data)
	if err != nil {
		return fmt.Errorf("ClientCodec.ReadResponseBody unzip error: %v", err)
	}

	return cc.s.Unmarshal(unzipData, body)
}

func (cc *ClientCodec) WriteRequest(req *rpc.Request, body interface{}) error {
	head := header.GetRequest()
	defer header.FreeRequest(head)

	cc.mtx.Lock()
	cc.pending[req.Seq] = req.ServiceMethod
	cc.mtx.Unlock()

	compressor, ok := compress.Compressors[cc.CompressType]
	if !ok {
		return fmt.Errorf("ClientCodec.WriteRequest error: not found compressType: %v", cc.CompressType)
	}

	data, err := cc.s.Marshal(body)
	if err != nil {
		return fmt.Errorf("ClientCodec.WriteRequest: %v", err)
	}

	zipData, err := compressor.Zip(data)
	if err != nil {
		return fmt.Errorf("ClientCodec.WriteRequest: %v", err)
	}

	head.CompressType = cc.CompressType
	head.MethodLen = uint16(len(req.ServiceMethod))
	head.ServiceMethod = req.ServiceMethod
	head.Seq = req.Seq
	head.RequestLen = uint32(len(zipData))

	head.Write(cc.w)
	cc.w.Write(zipData)
	return cc.w.Flush()
}

func (cc *ClientCodec) Close() error {
	return cc.c.Close()
}
