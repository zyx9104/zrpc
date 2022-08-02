package registry

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spaolacci/murmur3"
)

type Registry struct {
	ServiceMap map[string][]*service `json:"service_map"`
	UpdateTime map[string]time.Time  `json:"update_time"`
}

type service struct {
	Name string `json:"name"`
	Addr string `json:"addr"`
	Hash uint32 `json:"hash"`
}

type ByHash []*service

func (a ByHash) Len() int           { return len(a) }
func (a ByHash) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByHash) Less(i, j int) bool { return a[i].Hash < a[j].Hash }

var R = &Registry{ServiceMap: map[string][]*service{}, UpdateTime: make(map[string]time.Time)}

func (r *Registry) Put(ServiceName, addr string) {
	ss := r.ServiceMap[ServiceName]
	f := false
	for i := 0; i < len(ss); i++ {
		if ss[i].Addr == addr {
			f = true
			break
		}
	}
	if !f {
		ss = append(ss, &service{Name: ServiceName, Addr: addr, Hash: murmur3.Sum32([]byte(addr))})
	}
	sort.Sort(ByHash(ss))
	r.ServiceMap[ServiceName] = ss
	r.UpdateTime[ServiceName] = time.Now()
}

func (r *Registry) Get(ServiceName string) []*service {

	return r.ServiceMap[ServiceName]
}

func (r *Registry) Del(ServiceName, addr string) {
	ss := r.ServiceMap[ServiceName]
	idx := -1
	for i := 0; i < len(ss); i++ {
		if ss[i].Addr == addr {
			idx = i
			break
		}
	}
	if idx == -1 {
		return
	}
	r.ServiceMap[ServiceName] = append(ss[:idx], ss[idx+1:]...)
	r.UpdateTime[ServiceName] = time.Now()
}

type Request struct {
	ServiceName string `json:"service_name,omitempty"`
	Addr        string `json:"addr,omitempty"`
}

type Response struct {
	Service    []*service `json:"service,omitempty"`
	UpdateTime time.Time  `json:"update_time,omitempty"`
}

func Register(ctx *gin.Context) {
	req := &Request{}
	ctx.BindJSON(req)

	R.Put(req.ServiceName, req.Addr)

	ctx.JSON(http.StatusOK, gin.H{
		"msg": "success",
	})
}

func Delete(ctx *gin.Context) {
	req := &Request{}
	ctx.BindJSON(req)
	R.Del(req.ServiceName, req.Addr)
	ctx.JSON(http.StatusOK, gin.H{
		"msg": "success",
	})
}

func Get(ctx *gin.Context) {
	req := &Request{}
	ctx.BindJSON(req)

	resp := &Response{Service: R.Get(req.ServiceName)}
	ctx.JSON(http.StatusOK, resp)
}

func Init(e *gin.Engine) {
	e.POST("/get", Register)
	e.POST("/register", Delete)
	e.POST("/delete", Get)
}
