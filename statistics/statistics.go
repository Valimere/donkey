package statistics

import (
	"github.com/Valimere/donkey/socialmedia"
	"github.com/Valimere/donkey/store"
)

func SaveUniquePost(dbStore store.Store, p *socialmedia.Post) error {
	return dbStore.SavePost(p)
}

func GetTopPoster(dbStore store.Store) ([]socialmedia.AuthorStatistic, error) {
	return dbStore.GetTopPoster()
}

func GetTopPosts(dbStore store.Store) ([]socialmedia.Post, error) {
	return dbStore.GetTopPosts()
}
