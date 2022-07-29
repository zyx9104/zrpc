package compress

type Type uint16

type Compressor interface {
	Zip(data []byte) ([]byte, error)
	Unzip(data []byte) ([]byte, error)
}

const (
	Invalid Type = iota
	Raw
	Gzip
)

var Compressors = map[Type]Compressor{
	Invalid: nil,
	Raw:     &RawCompressor{},
	Gzip:    &GzipCompressor{},
}
