package zrpc

import (
	"errors"
	"hash/crc32"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Registry interface {
	Register(serviceName, addr string) error
	LookUp(serviceName string) ([]serverItem, error)
	Listen(addr string)
}

func NewRegistry() Registry {
	return &TinyRegistry{
		servers: make(map[string][]serverItem),
	}
}

type TinyRegistry struct {
	mtx     sync.Mutex
	servers map[string][]serverItem
}

type serverItem struct {
	Addr       string    `json:"addr"`
	Hash       uint32    `json:"hash"`
	UpdateTime time.Time `json:"updateTime"`
}

func (r *TinyRegistry) Register(serviceName, addr string) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	st := serverItem{Addr: addr, UpdateTime: time.Now(), Hash: crc32.ChecksumIEEE([]byte(addr))}
	ss := r.servers[serviceName]
	f := false
	for i := 0; i < len(ss); i++ {
		if ss[i].Hash == st.Hash {
			f = true
			ss[i] = st
		}
	}
	if !f {
		ss = append(ss, st)
		r.servers[serviceName] = ss
	}
	return nil
}

func (r *TinyRegistry) LookUp(serviceName string) (addrs []serverItem, err error) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	return r.servers[serviceName], err
}

func (r *TinyRegistry) Listen(addr string) {
	g := gin.Default()
	g.GET("/register", r.register)
	g.GET("/lookup", r.lookup)
	g.Run(addr)
}

func (r *TinyRegistry) register(ctx *gin.Context) {
	serviceName := ctx.Query("serviceName")
	addr := ctx.Query("addr")
	err := r.Register(serviceName, addr)
	log.Print("path args:", serviceName, addr)
	if err == nil {
		err = errors.New("ok")
	}
	ctx.JSON(http.StatusOK, gin.H{"msg": err})
}

func (r *TinyRegistry) lookup(ctx *gin.Context) {
	serviceName := ctx.Query("serviceName")
	servers, err := r.LookUp(serviceName)
	log.Print("servers: ", servers)

	ctx.JSON(http.StatusOK, gin.H{"msg": err, "data": servers})
}
