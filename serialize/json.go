package serialize

type JsonSerializer struct {
}

func (s *JsonSerializer) Marshal(body interface{}) ([]byte, error) {
	return nil, nil
}
func (s *JsonSerializer) Unmarshal(data []byte, body interface{}) error {
	return nil
}
