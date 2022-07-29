package zrpc

import (
	"io"
	"net/rpc"

	"github.com/z-y-x233/zrpc/codec"
	"github.com/z-y-x233/zrpc/compress"
	"github.com/z-y-x233/zrpc/serialize"
)

type Client struct {
	*rpc.Client
}

type Options struct {
	SerializeType serialize.Type
	CompressType  compress.Type
}

// NewClient Create a new rpc client
func NewClient(conn io.ReadWriteCloser, opt *Options) *Client {

	return &Client{rpc.NewClientWithCodec(
		codec.NewClientCodec(conn, serialize.Serializers[opt.SerializeType], opt.CompressType))}
}

// Call synchronously calls the rpc function
func (c *Client) Call(serviceMethod string, args interface{}, reply interface{}) error {
	return c.Client.Call(serviceMethod, args, reply)
}

// AsyncCall asynchronously calls the rpc function and returns a channel of *rpc.Call
func (c *Client) AsyncCall(serviceMethod string, args interface{}, reply interface{}) chan *rpc.Call {
	return c.Go(serviceMethod, args, reply, nil).Done
}
