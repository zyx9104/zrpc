package header

import "sync"

var (
	reqs  = sync.Pool{New: func() interface{} { return new(RequestHeader) }}
	resps = sync.Pool{New: func() interface{} { return new(ResponseHeader) }}
)

func GetRequest() *RequestHeader {
	return reqs.Get().(*RequestHeader)
}

func FreeRequest(req *RequestHeader) {
	reqs.Put(req)
}

func GetResponse() *ResponseHeader {
	return resps.Get().(*ResponseHeader)

}
func FreeResponse(resp *ResponseHeader) {
	resps.Put(resp)
}
