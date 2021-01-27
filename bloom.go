package BloomFilter

import (
	"errors"

	"github.com/go-redis/redis/v8"
)

type Bloom interface {
	Add(string) error
	IsExist(string) bool
	Clear()
}

func NewBloom(typ, key string) (Bloom, error) {
	switch typ {
	case "redis":
		redisCli := redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "", // no password set
			DB:       0,  // use default DB
		})
		return NewRedisBloom(redisCli, key), nil
	default:
		return nil, errors.New("the type is not valid")
	}
}
