package header

import (
	"bufio"
	"encoding/binary"
	"fmt"

	"github.com/z-y-x233/zrpc/compress"
)

const HeaderSize = 16

type RequestHeader struct {
	MethodLen     uint16
	CompressType  compress.Type
	RequestLen    uint32
	Seq           uint64
	ServiceMethod string
}

func (req *RequestHeader) Read(rd *bufio.Reader) error {
	l, err := rd.Peek(2)
	if err != nil {
		return err
	}

	req.MethodLen = binary.BigEndian.Uint16(l)
	data := make([]byte, HeaderSize+req.MethodLen)
	n, err := rd.Read(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("RequestHeader.Read error: read %v bytes, should %v byte", n, len(data))
	}

	req.CompressType = compress.Type(binary.BigEndian.Uint16(data[2:]))
	req.RequestLen = binary.BigEndian.Uint32(data[4:])
	req.Seq = binary.BigEndian.Uint64(data[8:])
	req.ServiceMethod = string(data[16:])
	return nil
}

func (req *RequestHeader) Write(wt *bufio.Writer) error {
	data := make([]byte, 16+len(req.ServiceMethod))
	binary.BigEndian.PutUint16(data, req.MethodLen)
	binary.BigEndian.PutUint16(data[2:], uint16(req.CompressType))
	binary.BigEndian.PutUint32(data[4:], req.RequestLen)
	binary.BigEndian.PutUint64(data[8:], req.Seq)
	n := copy(data[16:], req.ServiceMethod)
	if n != int(req.MethodLen) {
		return fmt.Errorf("RequestHeader.write error: write %v bytes, should %v byte", n, req.MethodLen)
	}
	wt.Write(data)
	return wt.Flush()
}

type ResponseHeader struct {
	CompressType compress.Type
	ErrorLen     uint16
	ResponseLen  uint32
	Seq          uint64
	Error        string
}

func (resp *ResponseHeader) Read(rd *bufio.Reader) error {
	l, err := rd.Peek(2)
	if err != nil {
		return err
	}

	resp.ErrorLen = binary.BigEndian.Uint16(l)
	data := make([]byte, HeaderSize+resp.ErrorLen)
	n, err := rd.Read(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("ResponseHeader.Read error: read %v bytes, should %v byte", n, len(data))
	}

	resp.CompressType = compress.Type(binary.BigEndian.Uint16(data[2:]))
	resp.ResponseLen = binary.BigEndian.Uint32(data[4:])
	resp.Seq = binary.BigEndian.Uint64(data[8:])
	resp.Error = string(data[16:])
	return nil
}

func (resp *ResponseHeader) Write(wt *bufio.Writer) error {
	data := make([]byte, 16+len(resp.Error))
	binary.BigEndian.PutUint16(data, resp.ErrorLen)
	binary.BigEndian.PutUint16(data[2:], uint16(resp.CompressType))
	binary.BigEndian.PutUint32(data[4:], resp.ResponseLen)
	binary.BigEndian.PutUint64(data[8:], resp.Seq)
	n := copy(data[16:], resp.Error)
	if n != int(resp.ErrorLen) {
		return fmt.Errorf("ResponseHeader.write error: write %v bytes, should %v byte", n, resp.ErrorLen)
	}
	wt.Write(data)
	return wt.Flush()
}
