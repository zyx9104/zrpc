package header

import "sync"

type Request struct {
	Seq           uint64
	ServiceMethod string
}

type Response struct {
	Seq           uint64
	ServiceMethod string
	Error         string
}

var (
	reqPool  = sync.Pool{New: func() interface{} { return new(Request) }}
	respPool = sync.Pool{New: func() interface{} { return new(Response) }}
)

func GetRequest() *Request {
	return reqPool.Get().(*Request)
}

func FreeRequest(request *Request) {
	reqPool.Put(request)
}

func GetResponse() *Response {
	return respPool.Get().(*Response)
}

func FreeResponse(response *Response) {
	respPool.Put(response)

}
