package social

import (
	"context"
	"errors"
	"fmt"
	"strconv"
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
users:   hash ------  login -> id
status:$statusId  ---- status
	message:
	posted:
	id:
	uid:
	login:
status:id:  ---- int // status唯一ID

home:timeline: ----- zset
	$statusId    $posted

user:timeline:$uid ----- zset
	$statusId    $posted

followers:$uid  ----- zset
	$uid1   $time
following:$uid  ----- zset
	$uid1   $time

home:$uid  ------ zset
	$statusId   $posted


朋友圈的timeline

*/
func NewSocial() *Social {
	return &Social{Client: initRedisClient()}
}

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
	Posted  int
	Id      string
	UID     string
	Login   string
}

func createStatus(uid string, message string, login string, statusId string) *Status {
	return &Status{
		Message: message,
		Posted:  int(time.Now().Unix()),
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

func (status *Status) MTo(m map[string]string) error {
	if message, ok := m["message"]; ok {
		status.Message = message
	}
	if posted, ok := m["posted"]; ok {
		postedInt, err := strconv.Atoi(posted)
		if err != nil {
			return err
		}
		status.Posted = postedInt
	}
	if id, ok := m["id"]; ok {
		status.Id = id
	}
	if uid, ok := m["uid"]; ok {
		status.UID = uid
	}
	if login, ok := m["login"]; ok {
		status.Login = login
	}
	return nil
}

func (social *Social) NewStatus(uid string, message string) error {
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

// HomeTimeLine: 主页的timeline
func (social *Social) HomeTimeLine(size int64, offset int64) []*Status {
	statusIds := social.Client.ZRevRangeByScore(context.Background(), "home:timeline:", &redis.ZRangeBy{
		Offset: offset,
		Count:  size,
	})
	pipeline := social.Client.TxPipeline()
	for _, id := range statusIds.Val() {
		pipeline.HGetAll(context.Background(), id)
	}
	statusCmds, err := pipeline.Exec(context.Background())
	if err != nil {
		return nil
	}
	statuses := []*Status{}
	for _, cmd := range statusCmds {
		if cmd.Err() == nil {
			status := &Status{}
			err := status.MTo(cmd.(*redis.StringStringMapCmd).Val())
			if err != nil {
				statuses = append(statuses, status)
			}
		}
	}
	return statuses
}

func (social *Social) ProfileTimeline(uid string, size int64, offset int64) []*Status {
	statusIds := social.Client.ZRevRangeByScore(context.Background(), "home:timeline:"+uid, &redis.ZRangeBy{
		Offset: offset,
		Count:  size,
	})
	pipeline := social.Client.TxPipeline()
	for _, id := range statusIds.Val() {
		pipeline.HGetAll(context.Background(), id)
	}
	statusCmds, err := pipeline.Exec(context.Background())
	if err != nil {
		return nil
	}
	statuses := []*Status{}
	for _, cmd := range statusCmds {
		if cmd.Err() == nil {
			status := &Status{}
			err := status.MTo(cmd.(*redis.StringStringMapCmd).Val())
			if err != nil {
				statuses = append(statuses, status)
			}
		}
	}
	return statuses
}

const HOME_TIME_LINE_LIMIT = 1000

func (social *Social) FollowUser(fromId, toId string) error {
	// 插入follower following 表
	// 更新user info 表
	// 把toId的timeline 放到fromId的朋友圈timeline里面去（可以有一个上限 比如1000）
	pipeline := social.Client.TxPipeline()
	ctx := context.Background()
	now := float64(time.Now().Unix())
	pipeline.ZAdd(ctx, "followers:"+toId, &redis.Z{
		Score:  now,
		Member: fromId,
	})
	pipeline.ZAdd(ctx, "following:"+fromId, &redis.Z{
		Score:  now,
		Member: toId,
	})
	pipeline.ZRevRangeByScoreWithScores(ctx, "home:"+toId, &redis.ZRangeBy{
		Offset: 0,
		Count:  HOME_TIME_LINE_LIMIT,
	})
	fCmds, err := pipeline.Exec(ctx)
	if err != nil {
		return err
	}
	followerCount := fCmds[0].(*redis.IntCmd).Val()
	followingCount := fCmds[1].(*redis.IntCmd).Val()
	timelineTo := fCmds[2].(*redis.ZSliceCmd).Val()
	ntimelineTo := []*redis.Z{}
	for _, e := range timelineTo {
		ntimelineTo = append(ntimelineTo, &e)
	}
	pipeline = social.Client.TxPipeline()
	pipeline.HIncrBy(ctx, "user:"+fromId, "followings", followingCount)
	pipeline.HIncrBy(ctx, "user:"+toId, "followers", followerCount)
	pipeline.ZAdd(ctx, "home:"+fromId, ntimelineTo...)
	pipeline.ZRemRangeByRank(ctx, "home:"+fromId, 0, -HOME_TIME_LINE_LIMIT)
	_, err = pipeline.Exec(ctx)
	return err
}

func (social *Social) UnFollowUser(fromId, toId string) error {
	// 删除follower following 表中对应的人
	// 更新user info表
	// 从fromId中把toId 的timeline 去除
	pipeline := social.Client.TxPipeline()
	ctx := context.Background()
	pipeline.ZRem(ctx, "followers:"+toId, fromId)
	pipeline.ZRem(ctx, "following:"+fromId, toId)
	pipeline.ZRange(ctx, "home:"+toId, 0, HOME_TIME_LINE_LIMIT)
	fCmds, err := pipeline.Exec(ctx)
	if err != nil {
		return err
	}
	followerCount := fCmds[0].(*redis.IntCmd).Val()
	followingCount := fCmds[1].(*redis.IntCmd).Val()
	timelineTo := fCmds[2].(*redis.StringSliceCmd).Val()
	ntimeline := []interface{}{}
	for _, e := range timelineTo {
		ntimeline = append(ntimeline, e)
	}
	pipeline = social.Client.TxPipeline()
	pipeline.HIncrBy(ctx, "user:"+fromId, "followings", -followingCount)
	pipeline.HIncrBy(ctx, "user:"+toId, "followers", -followerCount)
	pipeline.ZRem(ctx, "home:"+fromId, ntimeline...)
	_, err = pipeline.Exec(ctx)
	return err
}
