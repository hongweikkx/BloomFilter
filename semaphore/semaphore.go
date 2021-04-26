package semaphore

import (
	"context"
	"fmt"
	"time"

	"github.com/hongweikkx/BloomFilter/lock"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

func initRedisClient() *redis.Client {
	redisCli := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return redisCli
}

type Semaphore struct {
	Client *redis.Client
}

func NewSemaphore() *Semaphore {
	return &Semaphore{Client: initRedisClient()}
}

func getUnfairCounterSemaKey(key string) string {
	return "unfairSema:" + key
}
func (sema *Semaphore) AcquireUnfairCounterSemaphore(key string, limit int64) (string, error) {
	identifire := uuid.NewString()
	now := time.Now()
	expireTime := now.Add(-time.Second * 10)
	pipe := sema.Client.TxPipeline()
	pipe.ZRemRangeByScore(context.Background(), getUnfairCounterSemaKey(key), "0", fmt.Sprint(expireTime.UnixNano()))
	pipe.ZAdd(context.Background(), getUnfairCounterSemaKey(key), &redis.Z{
		Score:  float64(now.UnixNano()),
		Member: identifire,
	})
	pipe.ZRank(context.Background(), getUnfairCounterSemaKey(key), identifire)
	cmds, err := pipe.Exec(context.Background())
	if err != nil {
		return "", err
	}
	if cmds[2].(*redis.IntCmd).Val() >= limit {
		sema.Client.ZRem(context.Background(), getUnfairCounterSemaKey(key), identifire)
		return "", nil
	}
	return identifire, nil
}

func (sema *Semaphore) ReleaseUnfairCounterSemaphore(key string, identifire string) {
	sema.Client.ZRem(context.Background(), getUnfairCounterSemaKey(key), identifire)
}

func (sema *Semaphore) AcquireCounterSemaphoreWithLock(key string, limit int64) (string, bool) {
	locker := lock.NewLocker()
	idLock, b := locker.AcquireLock(key, 10*time.Second, 5*time.Second)
	if !b {
		return "", b
	}
	defer locker.ReleaseLock(key, idLock)
	idSema, err := sema.AcquireUnfairCounterSemaphore(key, limit)
	if err != nil {
		return "", false
	}
	return idSema, true
}

func (sema *Semaphore) ReleaseCounterSemaphoreWithLock(key string, identifire string) bool {
	locker := lock.NewLocker()
	idLock, b := locker.AcquireLock(key, 10*time.Second, 5*time.Second)
	if !b {
		return false
	}
	defer locker.ReleaseLock(key, idLock)
	sema.ReleaseUnfairCounterSemaphore(key, identifire)
	return true
}
