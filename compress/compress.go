package compress

type Type uint8

type Compressor interface {
	Zip(data []byte) ([]byte, error)
	Unzip(data []byte) ([]byte, error)
}

const (
	Invalid Type = iota
	Raw
	Gzip
)

var Compressors = map[Type]func() Compressor{
	Invalid: func() Compressor { return nil },
}
