package option

import (
	"time"

	"github.com/z-y-x233/zrpc/codec"
)

type Options struct {
	CodecType      codec.Type
	ConnectTimeout time.Duration // 0 means no limit
	HandleTimeout  time.Duration
}

func DefaultOptions() *Options {
	return &Options{
		CodecType:      codec.Gob,
		ConnectTimeout: time.Second * 3,
		HandleTimeout:  time.Second * 3,
	}
}
