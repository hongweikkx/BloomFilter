package BloomFilter

import (
	"context"

	"github.com/go-redis/redis/v8"
	ghf "github.com/hongweikkx/GeneralHashFunctions"
)

type HashFunc func(string) uint

var HashFuncList = []HashFunc{ghf.RSHash, ghf.JSHash, ghf.PJWHash, ghf.ELFHash}
var redisBloomKey = "redis-bloom-key"
var redisOffsetMax uint = 1 << 32

type RedisBloom struct {
	RedisCli *redis.Client
}

func NewRedisBloom() *RedisBloom {
	redisCli := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	redisBloom := &RedisBloom{RedisCli: redisCli}
	return redisBloom
}

func (redisBloom *RedisBloom) Add(str string) error {
	for _, f := range HashFuncList {
		ir := redisBloom.ValidBitOffset(f(str))
		cmd := redisBloom.RedisCli.SetBit(context.Background(), redisBloomKey, int64(ir), 1)
		if cmd.Err() != nil {
			return cmd.Err()
		}
	}
	return nil
}

func (redisBloom *RedisBloom) IsExist(str string) bool {
	for _, f := range HashFuncList {
		ir := redisBloom.ValidBitOffset(f(str))
		if redisBloom.RedisCli.GetBit(context.Background(), redisBloomKey, int64(ir)).Val() == 0 {
			return false
		}
	}
	return true
}

func (redisBloom *RedisBloom) Clear() {
	redisBloom.RedisCli.FlushAll(context.Background())
}

func (redisBloom *RedisBloom) ValidBitOffset(old uint) uint {
	return old % redisOffsetMax
}
