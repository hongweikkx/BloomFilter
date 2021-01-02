package BloomFilter

import (
	"errors"
)

type Bloom interface {
	Add(string) error
	IsExist(string) bool
	Clear()
}

func NewBloom(typ string) (Bloom, error) {
	switch typ {
	case "redis":
		return NewRedisBloom(), nil
	default:
		return nil, errors.New("the type is not valid")
	}
}
