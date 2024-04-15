package db

import (
	"github.com/Valimere/donkey/socialmedia"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"log"
	"os"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestMain sets up the testing environment
func TestMain(m *testing.M) {
	// Set up anything required before starting the test
	os.Exit(m.Run())
}

// setupTestDB initializes and returns an in-memory SQLite database for testing
func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:memdb1?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Could not open db: %v", err)
	}
	if err := db.AutoMigrate(&Token{}, &Post{}, &AuthorStatistic{}); err != nil {
		log.Fatalf("Could not migrate db: %v", err)
	}
	return db
}

func clearTables(db *gorm.DB) {
	db.Exec("DELETE FROM tokens")
	db.Exec("DELETE FROM posts")
	db.Exec("DELETE FROM author_statistics")
}

func TestSaveToken(t *testing.T) {
	db := setupTestDB()
	defer clearTables(db)

	store := DbStore{DB: db}
	token := &oauth2.Token{
		AccessToken: "access-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(24 * time.Hour),
	}

	err := store.SaveToken(token)
	assert.NoError(t, err)
}

func TestGetToken(t *testing.T) {
	db := setupTestDB()
	defer clearTables(db)

	store := DbStore{DB: db}
	expectedToken := &oauth2.Token{
		AccessToken: "access-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(24 * time.Hour),
	}

	// Save token first to retrieve later
	store.SaveToken(expectedToken)

	retrievedToken, err := store.GetToken()
	assert.NoError(t, err)
	assert.Equal(t, expectedToken.AccessToken, retrievedToken.AccessToken)
}

func TestSavePost(t *testing.T) {
	db := setupTestDB()
	defer clearTables(db)

	store := DbStore{DB: db}
	post := &socialmedia.Post{
		PostID:      "1",
		Author:      "test_user",
		SubReddit:   "test_sub",
		Title:       "Test Post",
		UpVotes:     100,
		NumComments: 10,
	}

	err := store.SavePost(post)
	assert.NoError(t, err)
}

func TestClearPosts(t *testing.T) {
	db := setupTestDB()
	defer clearTables(db)

	store := DbStore{DB: db}

	// Add a post to clear later
	store.SavePost(&socialmedia.Post{
		PostID: "1",
		Title:  "Sample Post",
	})

	// Clear posts
	err := store.ClearPosts()
	assert.NoError(t, err)

	// Verify posts are cleared
	var count int64
	db.Model(&Post{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestGetMultipleTopPosts(t *testing.T) {
	db := setupTestDB()
	defer clearTables(db)

	store := DbStore{DB: db}

	// Add some posts
	store.SavePost(&socialmedia.Post{
		PostID:  "1",
		UpVotes: 200,
	})
	store.SavePost(&socialmedia.Post{
		PostID:  "2",
		UpVotes: 200,
	})
	store.SavePost(&socialmedia.Post{
		PostID:  "3",
		UpVotes: 100,
	})

	topPosts, err := store.GetTopPosts()
	assert.NoError(t, err)
	assert.Len(t, topPosts, 2) // Expecting 2 posts with the same number of upvotes
}

func TestGetTopPost(t *testing.T) {
	db := setupTestDB()
	defer clearTables(db)

	store := DbStore{DB: db}

	expectedPost := &socialmedia.Post{
		PostID:  "1",
		UpVotes: 300,
	}
	store.SavePost(expectedPost)
	store.SavePost(&socialmedia.Post{
		PostID:  "2",
		UpVotes: 200,
	})

	topPosts, err := store.GetTopPosts()
	assert.NoError(t, err)
	assert.NotEmpty(t, topPosts)
	assert.Equal(t, expectedPost.PostID, topPosts[0].PostID)
	assert.Equal(t, expectedPost.UpVotes, topPosts[0].UpVotes)
}
