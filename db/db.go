package db

import (
	"encoding/json"
	"golang.org/x/oauth2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"time"
)

type Store interface {
	SaveToken(token *oauth2.Token) error
	GetToken() (*oauth2.Token, error)
}

type DBStore struct {
	DB *gorm.DB
}

type Token struct {
	gorm.Model
	OAuthData string
	ExpiresAt time.Time
}

// Post represents the schema for the "posts" table
type Post struct {
	gorm.Model
	PostID      string
	Author      string
	Subreddit   string
	Title       string
	UpVotes     int
	NumComments int
}

// AuthorStatistic represents the schema for the "author_statistics" table
type AuthorStatistic struct {
	gorm.Model
	Author        string
	TotalPosts    int
	TotalUpvotes  int
	TotalComments int
}

var DB *gorm.DB

func InitDB(debugFlag bool) {
	var err error
	DB, err = gorm.Open(sqlite.Open("donkey.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Error while connecting to the database: %s", err)
	}

	err = DB.AutoMigrate(&Token{}, &Post{}, &AuthorStatistic{})
	if err != nil {
		log.Fatalf("Error while migrating the database: %s", err)
	}
}

func (s *DBStore) SaveToken(token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}

	dbToken := &Token{
		OAuthData: string(data),
		ExpiresAt: token.Expiry,
	}
	result := s.DB.Save(dbToken)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *DBStore) GetToken() (*oauth2.Token, error) {
	var token Token
	err := s.DB.Order("created_at desc").First(&token).Error
	if err != nil {
		return nil, err
	}

	var oauthToken oauth2.Token
	err = json.Unmarshal([]byte(token.OAuthData), &oauthToken)
	if err != nil {
		return nil, err
	}

	return &oauthToken, nil
}
