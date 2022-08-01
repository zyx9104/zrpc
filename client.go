package zrpc

import (
	"encoding/json"
	"errors"
	"fmt"
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
}

type Call struct {
	Seq           uint64
	ServiceMethod string
	Error         string
	Args          interface{}
	Reply         interface{}
	Done          chan *Call
}

func NewClient() *Client {

	return &Client{
		pending: make(map[uint64]*Call),
	}
}

func (client *Client) Dial(network string, address string) {
	conn, err := net.Dial(network, address)
	if err != nil {
		log.Fatal(err)
	}
	opt := &option.Options{CodecType: codec.Gob}
	json.NewEncoder(conn).Encode(opt)
	client.codec = codec.ClientCodecs[opt.CodecType](conn)
	go client.input()
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

func (client *Client) Call(serviceMethod string, args, reply interface{}) error {
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
