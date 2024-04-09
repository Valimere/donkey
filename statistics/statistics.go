package statistics

import (
	"github.com/Valimere/donkey/db"
	"gorm.io/gorm"
)

func SaveUniquePost(postID, author string, upvotes, comments int) error {
	post := db.Post{PostID: postID, Author: author, UpVotes: upvotes, NumComments: comments}
	result := db.DB.FirstOrCreate(&post, post)

	if result.Error != nil {
		return result.Error
	}

	var statistic db.AuthorStatistic
	result = db.DB.Where("author = ?", author).First(&statistic)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// If the statistic record doesn't exist yet, create a new one.
			statistic := db.AuthorStatistic{
				Author:        author,
				TotalPosts:    1,
				TotalUpvotes:  upvotes,
				TotalComments: comments,
			}
			result = db.DB.Create(&statistic)
		}
	} else {
		// If the statistic record exists, increment corresponding values
		statistic.TotalPosts++
		statistic.TotalUpvotes += upvotes
		statistic.TotalComments += comments
		result = db.DB.Save(&statistic)
	}

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func GetTopPoster() (*[]db.AuthorStatistic, error) {
	var statistics []db.AuthorStatistic
	var maxCount db.AuthorStatistic

	// First, retrieve the maximum total posts (highest poster)
	err := db.DB.Order("total_posts desc").First(&maxCount).Error
	if err != nil {
		return nil, err
	}

	// Find all authors with the maximum total posts count (i.e., ties)
	err = db.DB.Where("total_posts = ?", maxCount.TotalPosts).Find(&statistics).Error
	if err != nil {
		return nil, err
	}

	return &statistics, nil
}

func GetTopPosts() (*[]db.Post, error) {
	var posts []db.Post
	var postWithMostUps db.Post

	// First, retrieve the post with the most upvotes
	err := db.DB.Order("up_votes desc").First(&postWithMostUps).Error
	if err != nil {
		return nil, err
	}

	// Find all posts with the same number of upvotes in case there are multiple
	err = db.DB.Where("up_votes = ?", postWithMostUps.UpVotes).Find(&posts).Error
	if err != nil {
		return nil, err
	}
	return &posts, nil
}
