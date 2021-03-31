package autocomplete

import (
	"context"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

/*
规定name 只能是a-Z的字母组成的字符串
abb{   abc  abc{
ab`    aba  aba{

name ---- zset
abac  0
abcd  0
dacd  0
*/

func initRedisClient() *redis.Client {
	redisCli := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return redisCli
}

type AutoComplete struct {
	Client *redis.Client
}

func NewAutoComplete() *AutoComplete {
	return &AutoComplete{Client: initRedisClient()}
}

func getPrefix(name string) string {
	if len(name) == 0 {
		return "`{"
	}
	bname := []rune(name)
	bnameL := len(bname)
	bname[bnameL-1] = bname[bnameL-1] - 1
	return string(bname) + "{"
}

func getSuffix(name string) string {
	return name + "{"
}

func (ac *AutoComplete) Add(name string) {
	ac.Client.ZAdd(context.Background(), "name", &redis.Z{
		Score:  0,
		Member: name,
	})
}

func (ac *AutoComplete) autoComplete(name string) []string {
	prefix := getPrefix(name) + uuid.NewString()
	suffix := getSuffix(name) + uuid.NewString()
	rets := []string{}
	ac.Client.ZAdd(context.Background(), "name",
		&redis.Z{Score: 0, Member: prefix},
		&redis.Z{Score: 0, Member: suffix})
	complete := func(tx *redis.Tx) error {
		preRank := tx.ZRank(context.Background(), "name", prefix).Val()
		sufRank := tx.ZRank(context.Background(), "name", suffix).Val()
		pipe := tx.TxPipeline()
		pipe.ZRange(context.Background(), "name", preRank, sufRank)
		pipe.ZRem(context.Background(), "name", prefix, suffix)
		cmds, err := pipe.Exec(context.Background())
		if err != nil {
			return err
		}
		rets = cmds[0].(*redis.StringSliceCmd).Val()
		return err
	}
	for i := 0; i < 5; i++ {
		err := ac.Client.Watch(context.Background(), complete, "name")
		if err == redis.TxFailedErr {
			continue
		}
		nrets := []string{}
		for _, ret := range rets {
			if !strings.Contains(ret, "{") {
				nrets = append(nrets, ret)
			}
		}
		return nrets
	}
	return rets
}
