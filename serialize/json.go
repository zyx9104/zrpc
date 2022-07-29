package serialize

import "encoding/json"

type JsonSerializer struct {
}

func (JsonSerializer) Marshal(body interface{}) ([]byte, error) {
	return json.Marshal(body)
}
func (JsonSerializer) Unmarshal(data []byte, body interface{}) error {
	return json.Unmarshal(data, body)
}
