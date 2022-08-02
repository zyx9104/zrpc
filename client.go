package zrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/z-y-x233/zrpc/codec"
	"github.com/z-y-x233/zrpc/header"
	"github.com/z-y-x233/zrpc/option"
)

type Client struct {
	codec   codec.ClientCodec
	sending sync.Mutex
	mtx     sync.Mutex
	Seq     uint64
	pending map[uint64]*Call
	opt     *option.Options
}

type Call struct {
	Seq           uint64
	ServiceMethod string
	Error         string
	Args          interface{}
	Reply         interface{}
	Done          chan *Call
}

func parseOptions(opt ...*option.Options) *option.Options {
	if len(opt) > 0 && opt[0] != nil {
		return opt[0]
	}
	return option.DefaultOptions()
}

func NewClientWithCodec(codec codec.ClientCodec) *Client {
	client := &Client{
		pending: make(map[uint64]*Call),
		codec:   codec,
	}
	go client.input()
	return client
}

func newClient(conn io.ReadWriteCloser, opt *option.Options) (*Client, error) {

	client := &Client{
		pending: make(map[uint64]*Call),
		opt:     opt,
	}
	err := json.NewEncoder(conn).Encode(client.opt)
	if err != nil {
		return nil, err
	}
	client.codec = codec.ClientCodecs[client.opt.CodecType](conn)
	if client.codec == nil {
		return nil, fmt.Errorf("codec %q not found", opt.CodecType)
	}
	return client, nil
}

func Dial(network string, address string, opt ...*option.Options) (client *Client) {
	client, err := dialTimeout(network, address, opt...)
	if err != nil {
		log.Fatal(err)
	}
	go client.input()
	return
}

func dialTimeout(network, address string, opts ...*option.Options) (client *Client, err error) {
	opt := parseOptions(opts...)
	var conn net.Conn
	if opt.ConnectTimeout == 0 {
		conn, err = net.Dial(network, address)
	} else {
		conn, err = net.DialTimeout(network, address, opt.ConnectTimeout)
	}
	if err != nil {
		return nil, err
	}
	client, err = newClient(conn, opt)
	if err != nil {
		return nil, err
	}
	return
}

func (client *Client) registerCall(call *Call) {
	client.mtx.Lock()
	defer client.mtx.Unlock()
	call.Seq = client.Seq
	client.pending[call.Seq] = call
	client.Seq++

}

func (client *Client) deleteCall(Seq uint64) *Call {
	client.mtx.Lock()
	defer client.mtx.Unlock()
	call := client.pending[Seq]
	delete(client.pending, Seq)
	return call
}

func (client *Client) Call(serviceMethod string, args, reply interface{}) (err error) {
	call := <-client.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
	return errors.New(call.Error)
}

func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Fatal("client.go: chan not buffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	go client.send(call)
	return call
}

func (client *Client) send(call *Call) {
	client.registerCall(call)
	req := header.GetRequest()
	req.Seq = call.Seq
	req.ServiceMethod = call.ServiceMethod

	client.sending.Lock()
	err := client.codec.WriteRequest(req, call.Args)
	client.sending.Unlock()

	if err != nil {
		log.Fatal("send Call error: ", err)
	}
}

func (client *Client) input() {
	var err error = nil
	for err == nil {
		resp := header.GetResponse()
		if err = client.codec.ReadResponseHeader(resp); err != nil {
			log.Print("clien input read header: ", err)
			break
		}
		if resp.Error != "" {
			log.Print("client resp err: ", resp.Error)
			break
		}

		call := client.deleteCall(resp.Seq)
		switch {
		case call == nil:
			err = fmt.Errorf("method: %v, seq %v not found", resp.ServiceMethod, resp.Seq)
			log.Print(err)
			err = client.codec.ReadResponseBody(nil)
		case call.Error != "":
			err = errors.New(call.Error)
			log.Print(err)
			err = client.codec.ReadResponseBody(nil)
			call.done()
		default:
			err = client.codec.ReadResponseBody(call.Reply)
			call.done()
		}

		header.FreeResponse(resp)
	}
	client.mtx.Lock()
	for _, c := range client.pending {
		c.Error = err.Error()
		delete(client.pending, c.Seq)
	}
	client.mtx.Unlock()

}

func (c *Call) done() {
	c.Done <- c
}
