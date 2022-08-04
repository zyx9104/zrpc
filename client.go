package zrpc

import (
	"errors"
	"io"
	"log"
	"sync"
)

type ClientCodec interface {
	ReadResponseHeader(resp *Response) error
	ReadResponseBody(body any) error
	WriteRequest(req *Request, body any) error
	Close() error
}

type Call struct {
	ServiceMethod string
	Seq           uint64
	Args          any
	Reply         any
	Error         error
	Done          chan *Call
}

type Client struct {
	codec ClientCodec

	reqMutex sync.Mutex // protects following
	request  *Request

	mutex    sync.Mutex // protects following
	seq      uint64
	pending  map[uint64]*Call
	closing  bool // user has called Close
	shutdown bool // server has told us to stop
}

func defaultClientCodec(conn io.ReadWriteCloser) ClientCodec {
	return nil
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

func (c *Client) Call(serviceMethod string, argv, reply any) error {
	return (<-c.Go(serviceMethod, argv, reply, make(chan *Call, 1)).Done).Error
}

func (c *Client) Go(serviceMethod string, argv, reply any, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          argv,
		Reply:         reply,
		Done:          done,
	}

	c.mutex.Lock()
	c.seq++
	call.Seq = c.seq
	c.pending[call.Seq] = call
	c.mutex.Unlock()

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
	err := c.codec.WriteRequest(c.request, call.Args)
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
