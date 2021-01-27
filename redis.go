package BloomFilter

import (
	"context"

	"github.com/go-redis/redis/v8"
	ghf "github.com/hongweikkx/GeneralHashFunctions"
)

type HashFunc func(string) uint

var HashFuncList = []HashFunc{ghf.RSHash, ghf.JSHash, ghf.PJWHash, ghf.ELFHash}
var redisOffsetMax uint = 1 << 32

type RedisBloom struct {
	redisCli *redis.Client
	key      string
}

func NewRedisBloom(client *redis.Client, key string) *RedisBloom {
	redisBloom := &RedisBloom{redisCli: client, key: key}
	return redisBloom
}

func (redisBloom *RedisBloom) Add(str string) error {
	for _, f := range HashFuncList {
		ir := redisBloom.ValidBitOffset(f(str))
		cmd := redisBloom.redisCli.SetBit(context.Background(), redisBloom.key, int64(ir), 1)
		if cmd.Err() != nil {
			return cmd.Err()
		}
	}
	return nil
}

func (redisBloom *RedisBloom) IsExist(str string) bool {
	for _, f := range HashFuncList {
		ir := redisBloom.ValidBitOffset(f(str))
		if redisBloom.redisCli.GetBit(context.Background(), redisBloom.key, int64(ir)).Val() == 0 {
			return false
		}
	}
	return true
}

func (redisBloom *RedisBloom) Clear() {
	redisBloom.redisCli.Del(context.Background(), redisBloom.key)
}

func (redisBloom *RedisBloom) ValidBitOffset(old uint) uint {
	return old % redisOffsetMax
}
