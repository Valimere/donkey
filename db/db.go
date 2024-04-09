package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Valimere/donkey/socialmedia"
	"github.com/Valimere/donkey/store"
	"golang.org/x/oauth2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"strings"
	"time"
)

type DbStore struct {
	DB *gorm.DB
}

// Ensure DBStore implements store.Store
var _ store.Store = &DbStore{}

type Token struct {
	gorm.Model
	OAuthData string
	ExpiresAt time.Time
}

// Post represents the schema for the "posts" table
type Post struct {
	gorm.Model
	PostID      string `gorm:"unique"`
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

var db *gorm.DB

func InitDB(debugFlag bool) (*gorm.DB, error) {
	var err error
	var logLevel logger.LogLevel

	if debugFlag {
		logLevel = logger.Info
	} else {
		logLevel = logger.Silent
	}
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logLevel,
		})
	db, err = gorm.Open(sqlite.Open("donkey.db"), &gorm.Config{Logger: newLogger})
	if err != nil {
		log.Fatalf("Error while connecting to the database: %s", err)
	}

	err = db.AutoMigrate(&Token{}, &Post{}, &AuthorStatistic{})
	if err != nil {
		log.Fatalf("Error while migrating the database: %s", err)
	}

	return db, nil
}

func (s *DbStore) SaveToken(token *oauth2.Token) error {
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

func (s *DbStore) GetToken() (*oauth2.Token, error) {
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

func (s *DbStore) TransformToDBPost(p *socialmedia.Post) *Post {
	return &Post{
		Model:       gorm.Model{},
		PostID:      p.ID,
		Author:      p.Author,
		Subreddit:   p.SubReddit,
		Title:       p.Title,
		UpVotes:     p.UpVotes,
		NumComments: p.NumComments,
	}
}

func (s *DbStore) SavePost(p *socialmedia.Post) error {
	dbPost := s.TransformToDBPost(p)
	result := s.DB.Save(dbPost)
	if result.Error != nil {
		// no need to print sqlite post is not unique info
		if strings.Contains(result.Error.Error(), "UNIQUE constraint failed") {
			return nil
		}
		return result.Error
	}
	return s.SaveAuthorStatistic(p) // tightly coupling the two but is efficient for our current use case
}

func (s *DbStore) SaveAuthorStatistic(p *socialmedia.Post) error {
	var dbAuthorStatistic AuthorStatistic

	// Start a transaction
	tx := s.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if the author statistic already exists
	result := tx.Where("author = ?", p.Author).First(&dbAuthorStatistic)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		dbAuthorStatistic = AuthorStatistic{
			Author:        p.Author,
			TotalPosts:    1,
			TotalUpvotes:  p.UpVotes,
			TotalComments: p.NumComments,
		}
		fmt.Printf("New Post found, PostID: %s, Upvotes: %4d Comments: %4d, Author: %24s, Title: %24s\n",
			p.ID, p.UpVotes, p.NumComments, p.Author, p.Title)
		result = tx.Create(&dbAuthorStatistic)
	} else if result.Error != nil {
		tx.Rollback()
		return result.Error
	} else {
		dbAuthorStatistic.TotalPosts++
		dbAuthorStatistic.TotalUpvotes += p.UpVotes
		dbAuthorStatistic.TotalComments += p.NumComments
		result = tx.Save(&dbAuthorStatistic)
	}

	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}

	return tx.Commit().Error
}

func (s *DbStore) ClearPosts() error {
	return s.DB.Exec("DELETE FROM posts").Error
}

func (s *DbStore) ClearAuthorStatistics() error {
	return s.DB.Exec("DELETE FROM author_statistics").Error
}

func (s *DbStore) GetTopPoster() ([]socialmedia.AuthorStatistic, error) {
	var topPosters []socialmedia.AuthorStatistic
	var firstTopPoster socialmedia.AuthorStatistic

	// First, retrieve the maximum total posts (highest poster)
	err := s.DB.Order("total_posts desc").First(&firstTopPoster).Error
	if err != nil {
		return nil, err
	}

	// Find all autheors with the same maximum total posts (i.e. ties)
	err = s.DB.Where("total_posts = ?", firstTopPoster.TotalPosts).Find(&topPosters).Error
	if err != nil {
		return nil, err
	}

	return topPosters, nil

}
func (s *DbStore) GetTopPosts() ([]socialmedia.Post, error) {
	var posts []socialmedia.Post
	var postWithMostUps socialmedia.Post

	// First, retrieve the post with the most upvotes
	err := s.DB.Order("up_votes desc").First(&postWithMostUps).Error
	if err != nil {
		return nil, err
	}

	// Find all posts with the same number of upvotes in case there are multiple
	err = s.DB.Where("up_votes = ?", postWithMostUps.UpVotes).Find(&posts).Error
	if err != nil {
		return nil, err
	}
	return posts, nil
}
