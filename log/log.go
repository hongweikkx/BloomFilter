package log

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

/*
	1. 最新日志
    2. 常见日志
*/

func initRedisClient() *redis.Client {
	redisCli := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return redisCli
}

type Log struct {
	Client *redis.Client
}

func NewLog() *Log {
	return &Log{Client: initRedisClient()}
}

func (log *Log) getRecentLogKey(name, level string) string {
	return fmt.Sprintf("recently:%s:%s", name, level)
}

func (log *Log) RecentLog(name, message string, level string) {
	key := log.getRecentLogKey(name, level)
	pipeline := log.Client.Pipeline()
	pipeline.LPush(context.Background(), key, message)
	pipeline.LTrim(context.Background(), key, 0, 99)
	pipeline.Exec(context.Background())
}

func (log *Log) getCommonLogKey(name, level string) string {
	return fmt.Sprintf("%s:%s", name, level)
}

func (log *Log) getCommonLogStartKey(name, level string) string {
	return log.getCommonLogKey(name, level) + ":start"
}

func getDate() int {
	return int(time.Now().Unix() / 3600)
}

const RETRIES = 5

func (log *Log) CommonLog(name, message string, level string) error {
	txf := func(tx *redis.Tx) error {
		pipeline := tx.TxPipeline()
		nowHour := getDate()
		if tx.Exists(context.Background(), log.getCommonLogStartKey(name, level)).Val() == 0 {
			pipeline.Set(context.Background(), log.getCommonLogStartKey(name, level), nowHour, -1)
		} else {
			hour, _ := tx.Get(context.Background(), log.getCommonLogStartKey(name, level)).Int()
			if hour < nowHour {
				oldLogKey := log.getCommonLogKey(name, level)
				newLogKey := oldLogKey + ":last"
				pipeline.Rename(context.Background(), oldLogKey, newLogKey)
				oldStartKey := log.getCommonLogStartKey(name, level)
				newStartKey := oldStartKey + ":plast"
				pipeline.Rename(context.Background(), oldStartKey, newStartKey)
				pipeline.Set(context.Background(), log.getCommonLogStartKey(name, level), nowHour, -1)
			}
		}
		pipeline.ZIncrBy(context.Background(), log.getCommonLogKey(name, level), 1, message)
		_, err := pipeline.Exec(context.Background())
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i := 0; i < RETRIES; i++ {
		err := log.Client.Watch(ctx, txf, log.getCommonLogStartKey(name, level))
		if err == redis.TxFailedErr {
			continue
		}
		return err
	}
	return nil
}
