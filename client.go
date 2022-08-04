package zrpc

import (
	"bufio"
	"encoding/gob"
	"errors"
	"io"
	"log"
	"sync"
)

type ClientCodec interface {
	ReadResponseHeader(resp *Response) error
	ReadResponseBody(body interface{}) error
	WriteRequest(req *Request, body interface{}) error
	Close() error
}

type Call struct {
	ServiceMethod string
	Seq           uint64
	Args          interface{}
	Reply         interface{}
	Error         error
	Done          chan *Call
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

func defaultClientCodec(conn io.ReadWriteCloser) ClientCodec {
	return NewGobClientCodec(conn)
}

func NewClient(conn io.ReadWriteCloser) *Client {
	codec := defaultClientCodec(conn)

	c := &Client{
		codec:   codec,
		pending: make(map[uint64]*Call),
	}
	go c.input()
	return c
}

func NewClientWithCodec(codec ClientCodec) *Client {
	c := &Client{
		codec:   codec,
		pending: make(map[uint64]*Call),
	}
	go c.input()
	return c
}

func (c *Client) Call(serviceMethod string, argv, reply interface{}) error {
	call := <-c.Go(serviceMethod, argv, reply, make(chan *Call, 1)).Done
	return call.Error
}

func (c *Client) Go(serviceMethod string, argv, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          argv,
		Reply:         reply,
		Done:          done,
	}

	go c.send(call)
	return call
}

func (c *Client) input() {
	var err error
	for err == nil {
		response := &Response{}
		err = c.codec.ReadResponseHeader(response)
		if err != nil {
			break
		}

		c.mutex.Lock()
		call := c.pending[response.Seq]
		if call != nil {
			delete(c.pending, response.Seq)
		}
		c.mutex.Unlock()

		switch {
		case call == nil:

			err = c.codec.ReadResponseBody(nil)
			if err != nil {
				err = errors.New("reading error body: " + err.Error())
			}
		case response.Error != "":

			err = c.codec.ReadResponseBody(nil)
			if err != nil {
				err = errors.New("reading error body: " + err.Error())
			}
			call.done()
		default:
			err = c.codec.ReadResponseBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	log.Fatal(err)
}

func (c *Client) send(call *Call) {
	c.reqMutex.Lock()
	defer c.reqMutex.Unlock()
	// Register this call.
	c.mutex.Lock()
	if c.shutdown || c.closing {
		c.mutex.Unlock()
		call.Error = errors.New("client shutdonw")
		call.done()
		return
	}
	seq := c.seq
	c.seq++
	c.pending[seq] = call
	c.mutex.Unlock()

	// Encode and send the request.
	c.request.Seq = seq
	c.request.ServiceMethod = call.ServiceMethod
	err := c.codec.WriteRequest(&c.request, call.Args)

	if err != nil {
		c.mutex.Lock()
		call = c.pending[seq]
		delete(c.pending, seq)
		c.mutex.Unlock()
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

func (c *Call) done() {
	c.Done <- c
}

type gobClientCodec struct {
	rwc io.ReadWriteCloser
	enc *gob.Encoder
	dec *gob.Decoder
	buf *bufio.Writer
}

func NewGobClientCodec(conn io.ReadWriteCloser) *gobClientCodec {
	buf := bufio.NewWriter(conn)
	return &gobClientCodec{
		rwc: conn,
		enc: gob.NewEncoder(buf),
		dec: gob.NewDecoder(conn),
		buf: buf,
	}
}

func (cc *gobClientCodec) ReadResponseHeader(resp *Response) error {
	return cc.dec.Decode(resp)
}

func (cc *gobClientCodec) ReadResponseBody(body interface{}) error {
	return cc.dec.Decode(body)
}

func (cc *gobClientCodec) WriteRequest(req *Request, args interface{}) error {

	cc.enc.Encode(req)
	cc.enc.Encode(args)
	return cc.buf.Flush()
}

func (cc *gobClientCodec) Close() error {
	return cc.rwc.Close()
}
