package lock

import (
	"context"
	"errors"
	"time"

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

type Locker struct {
	Client *redis.Client
}

func NewLocker() *Locker {
	return &Locker{Client: initRedisClient()}
}

func GetLockKey(key string) string {
	return "lock:" + key
}

func (locker *Locker) AcquireLock(key string, lockTimeout, timeout time.Duration) (string, bool) {
	endTime := time.Now().Add(timeout)
	id := uuid.NewString()
	for time.Now().Before(endTime) {
		ctx := context.Background()
		cmd := locker.Client.SetNX(ctx, GetLockKey(key), id, -1)
		if cmd.Err() != nil || cmd.Val() == false {
			cmdT := locker.Client.TTL(ctx, GetLockKey(key))
			if cmdT.Val() == -1 {
				locker.Client.Expire(ctx, GetLockKey(key), lockTimeout)
			}
		} else {
			locker.Client.Expire(ctx, GetLockKey(key), lockTimeout)
			return id, true
		}
		time.Sleep(time.Millisecond * 10)
	}
	return "", false
}

func (locker *Locker) ReleaseLock(key string, identifier string) error {
	release := func(tx *redis.Tx) error {
		cmd := tx.Get(context.Background(), GetLockKey(key))
		if cmd.Err() == redis.Nil {
			return nil
		}
		if cmd.Val() == identifier {
			pipe := tx.TxPipeline()
			pipe.Del(context.Background(), GetLockKey(key))
			_, err := pipe.Exec(context.Background())
			return err
		} else {
			return errors.New("release other lock")
		}
	}
	for {
		err := locker.Client.Watch(context.Background(), release, GetLockKey(key))
		if err == redis.TxFailedErr {
			continue
		}
		return err
	}
}
