package zrpc

import "net/rpc"

type Client struct {
	*rpc.Client
}

type Options struct {
}

func NewClient(o Options) *Client {
	return nil
}