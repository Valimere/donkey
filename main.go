package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io/ioutil"
	"net/http"
)

var redditOAuthConfig = &oauth2.Config{
	ClientID:     "YOUR_CLIENT_ID",
	ClientSecret: "YOUR_CLIENT_SECRET",
	RedirectURL:  "http://localhost:8080/callback", // Make sure this matches the redirect URI in your Reddit app settings
	Scopes:       []string{"read"},
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://www.reddit.com/api/v1/authorize",
		TokenURL: "https://www.reddit.com/api/v1/access_token",
	},
}

func getRedditClient(ctx context.Context, token *oauth2.Token) *http.Client {
	return redditOAuthConfig.Client(ctx, token)
}

func fetchSubredditPosts(ctx context.Context, client *http.Client, subreddit string) ([]byte, error) {
	resp, err := client.Get(fmt.Sprintf("https://oauth.reddit.com/r/%s/hot", subreddit))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func main() {
	ctx := context.Background()
	// Redirect the user to the authorization URL
	authURL := redditOAuthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser:\n%s\n", authURL)
	fmt.Print("Enter the authorization code: ")
	var authCode string
	fmt.Scan(&authCode)

	token, err := redditOAuthConfig.Exchange(ctx, authCode)
	if err != nil {
		panic(err)
	}

	// Exchange the authorization code for a token
	// This code should be obtained from the redirect URL query params after user authentication
	//token, err := redditOAuthConfig.Exchange(ctx, "YOUR_AUTHORIZATION_CODE")
	//if err != nil {
	//	panic(err)
	//}

	client := getRedditClient(ctx, token)

	subreddit := "news" // Specify the subreddit you want to fetch
	posts, err := fetchSubredditPosts(ctx, client, subreddit)
	if err != nil {
		panic(err)
	}

	// For demonstration, printing JSON response
	var jsonData map[string]interface{}
	json.Unmarshal(posts, &jsonData)
	fmt.Println(jsonData)
}
