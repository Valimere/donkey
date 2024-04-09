package store

import (
	"github.com/Valimere/donkey/socialmedia"
	"golang.org/x/oauth2"
)

type Store interface {
	SaveToken(token *oauth2.Token) error
	GetToken() (*oauth2.Token, error)
	SavePost(post *socialmedia.Post) error
	ClearPosts() error
	ClearAuthorStatistics() error
	GetTopPoster() ([]socialmedia.AuthorStatistic, error)
	GetTopPosts() ([]socialmedia.Post, error)
}
