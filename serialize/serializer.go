package serialize

type Type uint8

type Serializer interface {
	Marshal(body interface{}) ([]byte, error)
	Unmarshal(data []byte, body interface{}) error
}

const (
	Invalid Type = iota
	Gob
	Json
	Proto
)

var Serializers = map[Type]func() Serializer{
	Json: func() Serializer { return &JsonSerializer{} },
}
