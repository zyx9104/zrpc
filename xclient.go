package zrpc

import "net"

type XClient struct {
	c       *Client
	d       Discovery
	service string
}

func NewXClient(registryaddr, service string, mode int) *XClient {
	c := &XClient{service: service, d: &TinyDiscovery{registryAddr: registryaddr, mode: mode}}

	return c
}

func (c *XClient) Call(method string, args, reply interface{}) {
	addr := c.d.GetOne(c.service)
	conn, _ := net.Dial("tcp", addr)
	c.c = NewClient(conn)
	c.c.Call(c.service+"."+method, args, reply)
}
