package zrpc

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Discovery interface {
	Refresh(string)
	GetOne(string) string
}

type TinyDiscovery struct {
	registryAddr string
	mode         int
	idx          int
	server       []serverItem
}

func (t *TinyDiscovery) GetOne(service string) string {
	t.Refresh(service)
	switch t.mode {
	case 0:
		t.idx = (t.idx + 1) % len(t.server)
		return t.server[t.idx].Addr
	default:
		return t.server[0].Addr
	}
}

func (t *TinyDiscovery) Refresh(service string) {
	resp, err := http.Get(fmt.Sprintf("http://%s/lookup?serviceName=%s", t.registryAddr, service))
	if err != nil {
		log.Fatal(err)
	}

	// res, _ := ioutil.ReadAll(resp.Body)
	js := json.NewDecoder(resp.Body)
	res := &Result{}
	js.Decode(res)
	// log.Print(res)
	t.server = res.Data
}

type Result struct {
	Msg  string       `json:"msg"`
	Data []serverItem `json:"data"`
}
