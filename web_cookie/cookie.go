package web_cookie

import (
	"math"
	"strconv"
	"time"

	"github.com/go-redis/redis"
)

func initRedisClient() *redis.Client {
	redisCli := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return redisCli
}

type Cookie struct {
	Client *redis.Client
}

func NewCookie() *Cookie {
	return &Cookie{Client: initRedisClient()}
}

const Token2UserIdKey = "login"

// CheckToken: 尝试获取并返回令牌对应的用户Id
func (cookie *Cookie) CheckToken(token string) string {
	cmd := cookie.Client.HGet(Token2UserIdKey, token)
	if cmd.Err() != nil {
		return ""
	}
	return cmd.Val()
}

const RecentlyLoginKey = "recentlyLogin"
const ViewedSetKey = "view:" // 客户看了哪些商品

// UpdateToken: 更新令牌
func (cookie *Cookie) UpdateToken(token string, userId string, itemIds []string) {
	// 但是这个token 用无限增加呀？应该有个过期的设置吧。 比如设置成 login:token -> userId, 然后对这个key 设置expire time
	// fake 需求 不必太执着. 之所以不使用过期是因为想有RecentlyLoginNum这个limit
	cookie.Client.HSet(Token2UserIdKey, token, userId)
	now := time.Now().Unix()
	cookie.Client.ZAdd(RecentlyLoginKey, redis.Z{
		Score:  float64(now),
		Member: token,
	})
	// 为啥key都是基于token的 而不是基于唯一id的呢？ 有点奇怪.
	// fake需求，所以不必太执着
	itemZ := []redis.Z{}
	for _, itemId := range itemIds {
		itemZ = append(itemZ, redis.Z{
			Score:  float64(now),
			Member: itemId,
		})
	}
	cookie.Client.ZAdd(ViewedSetKey+token, itemZ...)
	if len(itemZ) > 0 {
		cookie.Client.ZRemRangeByRank(ViewedSetKey+token, 0, -26)
	}
}

const RecentlyLoginNum = 1000000

// ClearSessionDaemon: daemon 运行, 用于清空session
func (cookie *Cookie) ClearSessionDaemon() {
	for {
		cmd := cookie.Client.ZCard(RecentlyLoginKey)
		if cmd.Val() <= RecentlyLoginNum {
			time.Sleep(time.Second)
			continue
		}
		// for每次拿100个 不够就等1s
		endIndex := int64(math.Min(float64(cmd.Val()-RecentlyLoginNum), 100))
		tokens := cookie.Client.ZRange(RecentlyLoginKey, 0, endIndex)

		cookie.Client.HDel(Token2UserIdKey, tokens.Val()...)
		cookie.Client.ZRem(RecentlyLoginKey, tokens.Val())
		views := []string{}
		carts := []string{}
		for _, token := range tokens.Val() {
			views = append(views, ViewedSetKey+token)
			carts = append(carts, CartKey+token)
		}
		cookie.Client.Del(views...)
	}
}

const CartKey = "cart:"

// AddToCart: 加入到购物车
func (cookie *Cookie) AddToCart(token string, itemId string, count int) {
	key := CartKey + token
	cmd := cookie.Client.HGet(key, itemId)
	if count > 0 {
		cookie.Client.HIncrBy(key, itemId, int64(count))
	} else {
		if cmd.Val() != "" {
			oldCount, err := strconv.Atoi(cmd.Val())
			if err != nil {
				nCount := count - oldCount
				if nCount <= 0 {
					cookie.Client.HDel(key, itemId)
				} else {
					cookie.Client.HSet(key, itemId, nCount)
				}
			}
		}
	}
}

const PageCacheKey = "pageCache:"

// PageCacheGet: 获取网页缓存
func (cookie *Cookie) PageCacheGet(request string) string {
	return cookie.Client.Get(PageCacheKey + request).Val()
}

// PageCacheSet: 设置网页缓存
func (cookie *Cookie) PageCacheSet(request, page string) {
	cookie.Client.Set(PageCacheKey+request, page, time.Minute*30)
}
