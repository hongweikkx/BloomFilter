package log

import (
	"context"
	"fmt"
	"strconv"
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

func (log *Log) GetRecentLogKey(name, level string) string {
	return fmt.Sprintf("recently:%s:%s", name, level)
}

func (log *Log) RecentLog(name, message string, level string) {
	key := log.GetRecentLogKey(name, level)
	pipeline := log.Client.Pipeline()
	pipeline.LPush(context.Background(), key, message)
	pipeline.LTrim(context.Background(), key, 0, 99)
	pipeline.Exec(context.Background())
}

func (log *Log) GetCommonLogKey(name, level string, hour string) string {
	return fmt.Sprintf("hour-%s:%s:%s", hour, name, level)
}

func (log *Log) GetCommonLogOffsetKey() string {
	return fmt.Sprintln("hour-common")
}

func getDate() int {
	return int(time.Now().Unix() / 3600)
}

func (log *Log) GetCommonLogOffset() string {
	cmd := log.Client.Get(context.Background(), log.GetCommonLogOffsetKey())
	if cmd.Err() != nil {
		return strconv.Itoa(getDate())
	}
	return cmd.Val()
}

func (log *Log) SetCommonLogOffset() {
	log.Client.Set(context.Background(), log.GetCommonLogOffsetKey(), getDate(), -1)
}

const RETRIES = 5

func (log *Log) CommonLog(name, message string, level string) error {
	txf := func(tx *redis.Tx) error {
		pipeline := log.Client.TxPipeline()
		hourS := log.GetCommonLogOffset()
		houI, _ := strconv.Atoi(hourS)
		nowHour := getDate()
		if houI < nowHour {
			pipeline.Set(context.Background(), log.GetCommonLogOffsetKey(), nowHour, -1)
		}
		key := log.GetCommonLogKey(name, level, log.GetCommonLogOffset())
		pipeline.ZIncrBy(context.Background(), key, 1, message)
		_, err := pipeline.Exec(context.Background())
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i := 0; i < RETRIES; i++ {
		err := log.Client.Watch(ctx, txf, log.GetCommonLogOffsetKey())
		if err == redis.TxFailedErr {
			continue
		}
		return err
	}
	return nil
}
