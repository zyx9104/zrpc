package compress

type RawCompressor struct{}

func (RawCompressor) Zip(data []byte) ([]byte, error) {
	return data, nil
}

func (RawCompressor) Unzip(data []byte) ([]byte, error) {
	return data, nil
}
