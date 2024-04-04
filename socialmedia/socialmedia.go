package socialmedia

import (
	"context"
	"golang.org/x/oauth2"
)

type Post struct {
	ID     string
	Title  string
	Body   string
	Author string
}

type SocialMedia interface {
	StartServer(ctx context.Context) error
	ExchangeAuthCode(ctx context.Context) (*oauth2.Token, error)
	FetchPosts(ctx context.Context, subreddit string) ([]Post, error)
}
