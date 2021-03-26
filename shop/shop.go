package shop

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
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

type Shop struct {
	Client *redis.Client
}

func NewShop() *Shop {
	return &Shop{Client: initRedisClient()}
}

/*
1. 个人包裹
inventory:17:   -- set
ItemA.1
ItemB.1
ItemC.1

2. 商城
market --- zset
ItemA.17  |  2
ItemE.20  |  30

3. 个人信息
user:17   --- hset
id:
founds: 20
*/

func GetUserInventoryKey(userId string) string {
	return fmt.Sprintf("inventory:%s", userId)
}

func GetUserInfoKey(userId string) string {
	return fmt.Sprintf("user:%s", userId)
}

func GetMarketKey() string {
	return "market"
}

const MaxRetries = 5

func (shop *Shop) GetUserFounds(userId string) int {
	cmd := shop.Client.HGet(context.Background(), GetUserInfoKey(userId), "founds")
	founds, _ := strconv.Atoi(cmd.Val())
	return founds
}

func (shop *Shop) NewUser(userId string, founds int, itemId ...string) {
	shop.Client.HMSet(context.Background(), GetUserInfoKey(userId), map[string]interface{}{"id": userId, "founds": founds})
	shop.Client.SAdd(context.Background(), GetUserInventoryKey(userId), itemId)
}

func (shop *Shop) AddToShop(userId string, itemId string, price float64) error {
	txf := func(tx *redis.Tx) error {
		cmd := shop.Client.SIsMember(context.Background(), GetUserInventoryKey(userId), itemId)
		if !cmd.Val() {
			return nil
		}
		pipeline := tx.TxPipeline()
		pipeline.ZAdd(context.Background(), GetMarketKey(), &redis.Z{
			Score:  price,
			Member: itemId,
		})
		pipeline.SRem(context.Background(), GetUserInventoryKey(userId), itemId)
		_, err := pipeline.Exec(context.Background())
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i := 0; i < MaxRetries; i++ {
		err := shop.Client.Watch(ctx, txf, GetUserInventoryKey(userId))
		if err == redis.TxFailedErr {
			continue
		}
		return err
	}
	return errors.New("increment reached maximum number of retries")
}

func (shop *Shop) PurchaseItem(buyerId, itemId string) error {
	txf := func(tx *redis.Tx) error {
		itemSplit := strings.Split(itemId, ".")
		if len(itemSplit) != 2 {
			return errors.New("itemId not valid")
		}
		userId := itemSplit[1]
		priceCmd := shop.Client.ZScore(context.Background(), GetMarketKey(), itemId)
		if priceCmd.Err() == redis.Nil {
			return errors.New("key not valid")
		}
		founds := shop.GetUserFounds(buyerId)
		price := int64(priceCmd.Val())
		if int64(founds) < price {
			return errors.New("founds not enough")
		}
		pipeline := shop.Client.TxPipeline()
		pipeline.HIncrBy(context.Background(), GetUserInfoKey(buyerId), "founds", -price)
		pipeline.HIncrBy(context.Background(), GetUserInfoKey(userId), "founds", price)
		pipeline.SAdd(context.Background(), GetUserInventoryKey(buyerId), itemId)
		pipeline.ZRem(context.Background(), GetMarketKey(), itemId)
		_, err := pipeline.Exec(context.Background())
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i := 0; i < MaxRetries; i++ {
		err := shop.Client.Watch(ctx, txf, GetUserInfoKey(buyerId))
		if err == redis.TxFailedErr {
			continue
		}
		return err
	}
	return errors.New("increment reached maximum number of retries")
}
