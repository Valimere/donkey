package socialmedia

import (
	"context"
	"golang.org/x/oauth2"
	"time"
)

type RedditResponse struct {
	Before string
	After  string
	Posts  []Post
}
type Post struct {
	ID          string
	Title       string
	Body        string
	Author      string
	NumComments int
	Upvotes     int
	Created     time.Time
}

type SocialMedia interface {
	StartServer(ctx context.Context) error
	ExchangeAuthCode(ctx context.Context) (*oauth2.Token, error)
	FetchPosts(ctx context.Context, subreddit string) ([]Post, error)
}
