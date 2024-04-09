package socialmedia

import (
	"context"
	"golang.org/x/oauth2"
	"time"
)

type RedditResponse struct {
	Before    string
	After     string
	Subreddit string
	Posts     []Post
}
type Post struct {
	PostID      string
	Title       string
	Body        string
	Author      string
	NumComments int
	UpVotes     int
	Created     time.Time
	SubReddit   string
}

type AuthorStatistic struct {
	Author        string
	TotalPosts    int
	TotalUpvotes  int
	TotalComments int
}

type SocialMedia interface {
	StartServer(ctx context.Context) error
	ExchangeAuthCode(ctx context.Context) (*oauth2.Token, error)
	FetchPosts(ctx context.Context, subreddit string) ([]Post, error)
}
