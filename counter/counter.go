package counter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

func initRedisClient() *redis.Client {
	redisCli := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return redisCli
}

type Counter struct {
	Client *redis.Client
}

func NewCounter() *Counter {
	return &Counter{Client: initRedisClient()}
}

/*
count:1:hits  -----   hash
100000: 126
100001: 128

countList  ----- zset
1:hits    1
5:hits    5

*/

var PRECISION = []int{1, 5, 60, 300, 3600, 18000, 86400}

func (counter *Counter) GetCounterKey(slap int) string {
	return fmt.Sprintf("count:%d:hits", slap)
}

func (counter *Counter) GetCountListKey() string {
	return "countList"
}

// CounterIncr: 增加hit
func (counter *Counter) CounterIncr(incr int64) error {
	now := time.Now().Unix()
	pipeline := counter.Client.Pipeline()
	for _, slap := range PRECISION {
		pnow := (int(now) / slap) * slap
		pnowS := strconv.Itoa(pnow)
		counter.Client.HIncrBy(context.Background(), counter.GetCounterKey(slap), pnowS, incr)
		counter.Client.ZAdd(context.Background(), counter.GetCountListKey(), &redis.Z{
			Score:  float64(slap),
			Member: fmt.Sprintf("%d:hits", slap),
		})
	}
	_, err := pipeline.Exec(context.Background())
	return err
}
