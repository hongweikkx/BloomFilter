package blog

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
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

type Blog struct {
	Client *redis.Client
}

func NewBlog() *Blog {
	return &Blog{Client: initRedisClient()}
}

type Article struct {
	Id       string
	PostTime int
	Link     string
	UserId   string
}

//  GetArticleId: 获得文章自增ID
func (blog *Blog) getArticleId() string {
	cmd := blog.Client.Incr("articleId")
	return fmt.Sprintf("%d", cmd.Val())
}

//  getArticleKey: 获得文章redis key
func (blog *Blog) getArticleKey(articleId string) string {
	return fmt.Sprintf("article:%s", articleId)

}

func (blog *Blog) GetArticle(articleId string) (*Article, error) {
	cmd := blog.Client.HMGet(blog.getArticleKey(articleId), "postTime", "link", "userId")
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	postTime := cmd.Val()[0].(string)
	postTimeI, _ := strconv.Atoi(postTime)
	link := cmd.Val()[1].(string)
	userId := cmd.Val()[2].(string)
	return &Article{
		Id:       articleId,
		PostTime: postTimeI,
		Link:     link,
		UserId:   userId,
	}, nil
}

// PostArticle: 发布文章
func (blog *Blog) PostArticle(userId string, link string, group []string) error {
	id := blog.getArticleId()
	article := map[string]interface{}{
		"id":       id,
		"postTime": time.Now().Unix(),
		"link":     link,
		"userId":   userId,
	}
	cmd := blog.Client.HMSet(blog.getArticleKey(id), article)
	if cmd.Err() == nil {
		blog.Client.SAdd(blog.getVoteUsrKey(id), userId)
		blog.Client.Expire(blog.getVoteUsrKey(id), ONE_WEEK_SECONDS*time.Second)
		now := time.Now().Unix()
		blog.Client.ZAdd(blog.getVoteKey(), redis.Z{
			Score:  float64(now + ONE_DAY_SECONDS/ARTICLES_PER_DAY),
			Member: blog.getVoteScoreKey(id),
		})
		blog.Client.ZAdd(blog.getTimeKey(), redis.Z{
			Score:  float64(now),
			Member: blog.getVoteTimeKey(id),
		})
		blog.AddOrRemoveGroup(id, group, []string{})
	}
	return cmd.Err()
}

func (blog *Blog) getGroupKey(groupName string) string {
	return "group:" + groupName
}

func (blog *Blog) AddOrRemoveGroup(articleId string, addGroup []string, removeGroup []string) {
	for _, group := range addGroup {
		blog.Client.SAdd(blog.getGroupKey(group), blog.getArticleKey(articleId))
	}
	for _, group := range removeGroup {
		blog.Client.SRem(blog.getGroupKey(group), blog.getArticleKey(articleId))
	}
}

const ONE_WEEK_SECONDS = 604800
const ONE_DAY_SECONDS = 86400
const ARTICLES_PER_DAY = 200

func (blog *Blog) getVoteKey() string {
	return "vote"
}

func (blog *Blog) getTimeKey() string {
	return "time"
}

func (blog *Blog) getVoteNumKey(articleId string) string {
	return fmt.Sprintf("voteNum:%s", articleId)
}

func (blog *Blog) getVoteScoreKey(articleId string) string {
	return fmt.Sprintf("article:%s", articleId)
}

func (blog *Blog) getVoteTimeKey(articleId string) string {
	return fmt.Sprintf("article:%s", articleId)
}

func (blog *Blog) getArticleIdFromVoteScoreKey(key string) string {
	r := strings.Split(key, ":")
	return r[len(r)-1]
}

func (blog *Blog) getVoteUsrKey(articleId string) string {
	return fmt.Sprintf("voteUsers:%s", articleId)
}

// 给文章投票
func (blog *Blog) Vote(userId string, articleId string) error {
	// 期限检测
	postTime := blog.Client.HGet(blog.getArticleKey(articleId), "postTime").Val()
	fmt.Println("postTime:", blog.getArticleKey(articleId), postTime)
	postTimeInt, err := strconv.Atoi(postTime)
	if err != nil {
		return err
	}
	if time.Now().Unix()-int64(postTimeInt) > ONE_WEEK_SECONDS {
		return errors.New("article vote time expire")
	}
	cmd := blog.Client.SAdd(blog.getVoteUsrKey(articleId), userId)
	if cmd.Val() == 1 {
		blog.Client.ZIncrBy(blog.getVoteKey(), float64(ONE_DAY_SECONDS/ARTICLES_PER_DAY), blog.getVoteScoreKey(articleId))
		blog.Client.HIncrBy(blog.getVoteNumKey(articleId), "votes", 1)
	}
	return nil
}

const ARTICLE_PER_PAGE = 10

// GetArticleListByScore: 获得对应page的列表
func (blog *Blog) GetArticleListByScore(page int, key string) ([]*Article, error) {
	start := int64((page - 1) * ARTICLE_PER_PAGE)
	end := start + ARTICLE_PER_PAGE - 1
	cmd := blog.Client.ZRevRange(key, start, end)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	ret := []*Article{}
	for _, id := range cmd.Val() {
		articleId := blog.getArticleIdFromVoteScoreKey(id)
		article, err := blog.GetArticle(articleId)
		if err != nil {
			continue
		}
		ret = append(ret, article)
	}
	return ret, nil
}

//  GetArticleListByTime:
func (blog *Blog) GetArticleListByTime(page int) ([]*Article, error) {
	start := int64((page - 1) * ARTICLE_PER_PAGE)
	end := start + ARTICLE_PER_PAGE - 1
	cmd := blog.Client.ZRevRange(blog.getTimeKey(), start, end)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	ret := []*Article{}
	for _, id := range cmd.Val() {
		articleId := blog.getArticleIdFromVoteScoreKey(id)
		article, err := blog.GetArticle(articleId)
		if err != nil {
			continue
		}
		ret = append(ret, article)
	}
	return ret, nil
}

func (blog *Blog) getGroupScoreKey(group string, page int) string {
	key := fmt.Sprintf("score:%s:%d", group, page)
	return key
}

func (blog *Blog) GetArticleListByGroupScore(group string, page int) ([]*Article, error) {
	key := blog.getGroupScoreKey(group, page)
	if blog.Client.Exists(key).Val() == 0 {
		blog.Client.ZInterStore(key, redis.ZStore{
			Weights:   nil,
			Aggregate: "MAX",
		}, blog.getGroupKey(group), blog.getVoteKey())
		blog.Client.Expire(key, time.Minute*60)
	}
	return blog.GetArticleListByScore(page, key)
}
