package base_test

import (
	"context"
	"strconv"
	"sync"
	"testing"
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

var Client *redis.Client

func init() {
	Client = initRedisClient()
}

func TestList(t *testing.T) {
	ctx := context.Background()
	Client.FlushAll(ctx)
	listKey := "llkey"
	go func() {
		time.Sleep(time.Second)
		Client.RPush(ctx, listKey, "hello, world")
	}()
	s := Client.BLPop(ctx, 2*time.Second, listKey)
	if s.Err() != nil {
		t.Errorf("err:%s", s.Err())
	} else {
		t.Logf("%+v", s.Val())
	}
}

const PUBSUBChannel = "pchannel"

func TestPubSub(t *testing.T) {
	ctx := context.Background()
	Client.FlushAll(ctx)
	go Publisher(t, 5)
	pubSub := Client.Subscribe(context.Background(), PUBSUBChannel)
	for i := 1; i <= 4; i++ {
		msg, err := pubSub.ReceiveMessage(context.Background())
		if err != nil {
			t.Error("err:", err.Error())
			return
		}
		t.Log(msg.Payload)
		if msg.Payload != strconv.Itoa(i) {
			t.Error("payload err:", msg.Payload, i)
		}
		if i == 4 {
			err := pubSub.Unsubscribe(context.Background(), PUBSUBChannel)
			if err != nil {
				t.Error("err:", err.Error())
			}
		}
	}
}

func Publisher(t *testing.T, n int) {
	for i := 1; i <= n; i++ {
		time.Sleep(time.Second)
		err := Client.Publish(context.Background(), PUBSUBChannel, i).Err()
		if err != nil {
			t.Error("error:", err.Error())
		}
	}
}

func TestSort(t *testing.T) {
	ctx := context.Background()
	Client.FlushAll(ctx)

	key := "sortK"
	Client.RPush(ctx, key, 4, 7, 3, 5, 10, 8, 15)

	// sort
	cmd := Client.Sort(ctx, key, &redis.Sort{})
	if cmd.Err() != nil || len(cmd.Val()) != 7 || cmd.Val()[0] != "3" {
		t.Errorf("err:%+v, val:%+v", cmd.Err(), cmd.Val())
	}
	// sort.Alpha
	cmd = Client.Sort(ctx, key, &redis.Sort{Alpha: true})
	if cmd.Err() != nil || len(cmd.Val()) != 7 || cmd.Val()[0] != "10" {
		t.Errorf("err:%+v, val:%+v", cmd.Err(), cmd.Val())
	}

	// sort.By
	Client.HSet(ctx, "d-7", map[string]interface{}{"field": 5})
	Client.HSet(ctx, "d-3", map[string]interface{}{"field": 1})
	Client.HSet(ctx, "d-10", map[string]interface{}{"field": 9})
	cmd = Client.Sort(ctx, key, &redis.Sort{By: "d-*->field"})
	if cmd.Err() != nil || len(cmd.Val()) != 7 || cmd.Val()[6] != "10" {
		t.Errorf("err:%+v, val:%+v", cmd.Err(), cmd.Val())
	}
}

func Trans(t *testing.T) {
	key := "transK"
	ctx := context.Background()
	pipeline := Client.TxPipeline()
	pipeline.Incr(ctx, key)
	time.Sleep(time.Second)
	pipeline.IncrBy(ctx, key, -1)
	pipeline.Exec(ctx)
	cmd := Client.Get(ctx, key)
	if cmd.Err() != nil || cmd.Val() != "0" {
		t.Errorf("err:%+v, val:%+v", cmd.Err(), cmd.Err())
	}
}

func TestTrans(t *testing.T) {
	var wg sync.WaitGroup
	n := 3
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			Trans(t)
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestInfo(t *testing.T) {
	cmd := Client.Info(context.Background(), "server")
	if cmd.Err() != nil {
		t.Error("err:", cmd.Err())
	} else {
		t.Log("server:", cmd.Val())
	}
}
