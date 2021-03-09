package blog

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
)

func TestBlog(t *testing.T) {
	blogIns := NewBlog()
	initData(blogIns)
	voteData(blogIns)
	l, err := blogIns.GetArticleListByGroupScore("0", 1)
	if err != nil {
		t.Error("err:", err.Error())
	} else {
		t.Log("len:", len(l))
		for _, a := range l {
			t.Log(*a)
		}
	}
}

func initData(blog *Blog) {
	for i := 0; i < 54; i++ {
		userId := i % 10
		userIdS := fmt.Sprintf("userId:%03d", userId)
		link := fmt.Sprintf("http://localhost:8080/%03d", i)
		group := rand.Intn(2)
		err := blog.PostArticle(userIdS, link, []string{strconv.Itoa(group)})
		if err != nil {
			panic(err)
		}
	}
}

func voteData(blog *Blog) {
	for i := 0; i < 5; i++ {
		userId := rand.Intn(10)
		articleId := rand.Intn(54)
		fmt.Printf("user:%03d vote article:%d\n", userId, articleId)
		userIdS := fmt.Sprintf("userId:%03d", userId)
		articleIdS := fmt.Sprintf("%d", articleId)
		err := blog.Vote(userIdS, articleIdS)
		if err != nil {
			panic(err.Error())
		}
	}
}
