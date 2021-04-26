package social

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hongweikkx/BloomFilter/lock"
)

func initRedisClient() *redis.Client {
	redisCli := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return redisCli
}

type Social struct {
	Client *redis.Client
}

type User struct {
	Login      string
	Id         string
	Name       string
	Followers  int64
	Followings int64
	Posts      int64
	SignUp     int64
}

/*
user:$UserId   ----- hset // User Info
	login:      user.Login,
	id:         user.Id,
	name:       user.Name,
	followers:  user.Followers,
	followings: user.Followings,
	posts:      user.Posts,
	signup:     user.SignUp,
user:id:  ------ int   // user唯一ID
users:   ------  login -> id
status:$StatusId  ---- status
	message:
	posted:
	id:
	uid:
	login:
status:id:  ---- int // status唯一ID
*/

func createUser(login string, name string, id string) *User {
	return &User{
		Login:      login,
		Id:         id,
		Name:       name,
		Followers:  0,
		Followings: 0,
		Posts:      0,
		SignUp:     time.Now().Unix(),
	}
}
func (user *User) toM() map[string]interface{} {
	return map[string]interface{}{
		"login":      user.Login,
		"id":         user.Id,
		"name":       user.Name,
		"followers":  user.Followers,
		"followings": user.Followings,
		"posts":      user.Posts,
		"signup":     user.SignUp,
	}
}

func (social *Social) NewUser(login string, name string) (string, error) {
	locker := lock.NewLocker()
	llogin := strings.ToLower(login) // 都是小写
	lockKey := "lock:userId:" + llogin
	identifier, isLock := locker.AcquireLock(lockKey, 10*time.Second, 3*time.Second)
	if !isLock {
		return "", errors.New("lock not valid")
	}
	defer locker.ReleaseLock(lockKey, identifier)
	existCmd := social.Client.Exists(context.Background(), "users:"+llogin)
	if existCmd.Val() == 1 {
		return "", errors.New("user exist")
	}
	id := fmt.Sprintf("%d", social.Client.Incr(context.Background(), "user:id:").Val())
	user := createUser(llogin, name, id)
	pipeline := social.Client.TxPipeline()
	pipeline.HSet(context.Background(), llogin, id)
	pipeline.HMSet(context.Background(), "user:"+id, user.toM())
	_, err := pipeline.Exec(context.Background())
	return id, err
}

type Status struct {
	Message string
	Posted  int64
	Id      string
	UID     string
	Login   string
}

func createStatus(uid string, message string, login string, statusId string) *Status {
	return &Status{
		Message: message,
		Posted:  time.Now().Unix(),
		Id:      statusId,
		UID:     uid,
		Login:   login,
	}
}

func (status *Status) ToM() map[string]interface{} {
	return map[string]interface{}{
		"message": status.Message,
		"posted":  status.Posted,
		"id":      status.Id,
		"uid":     status.UID,
		"login":   status.Login,
	}
}

func (social Social) NewStatus(uid string, message string) error {
	loginCmd := social.Client.HGet(context.Background(), "user:"+uid, "login")
	if loginCmd.Err() != nil {
		return loginCmd.Err()
	}
	login := loginCmd.Val()
	statusId := fmt.Sprintf("%d", social.Client.Incr(context.Background(), "status:id:").Val())
	status := createStatus(uid, message, login, statusId)
	pipeline := social.Client.TxPipeline()
	pipeline.HMSet(context.Background(), "status:"+statusId, status.ToM)
	pipeline.HIncrBy(context.Background(), "user:"+uid, "posts", 1)
	_, err := pipeline.Exec(context.Background())
	return err
}
