package reddit

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Structs for JSON parsing - these should match the structure of the Reddit JSON response
type RedditResponse struct {
	Data struct {
		Children []struct {
			Data PostData `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type PostData struct {
	Title string `json:"title"`
}

func GetSubredditPosts(subreddit string) ([]PostData, error) {
	url := fmt.Sprintf("https://www.reddit.com/r/%s/.json", subreddit)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var redditResponse RedditResponse
	err = json.NewDecoder(resp.Body).Decode(&redditResponse)
	if err != nil {
		return nil, err
	}

	posts := make([]PostData, 0)
	for _, child := range redditResponse.Data.Children {
		posts = append(posts, child.Data)
	}

	return posts, nil
}
