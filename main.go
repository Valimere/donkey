package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	//REDDIT_USER_AGENT = os.Getenv("REDDIT_USER_AGENT")
	REDDIT_CLIENT_ID = os.Getenv("REDDIT_CLIENT_ID")
	REDDIT_SECRET    = os.Getenv("REDDIT_SECRET")
	//REDDIT_USERNAME   = os.Getenv("REDDIT_USERNAME")
	//REDDIT_PASSWORD   = os.Getenv("REDDIT_PASSWORD")
	DELAY = time.Duration(1)
)

var redditOAuthConfig = &oauth2.Config{
	ClientID:     REDDIT_CLIENT_ID,
	ClientSecret: REDDIT_SECRET,
	RedirectURL:  "http://localhost:8080", // Make sure this matches the redirect URI in your Reddit app settings
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
	resp, err := client.Get(fmt.Sprintf("https://oauth.reddit.com/r/%s/new", subreddit))
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
	_, err := fmt.Scan(&authCode)
	if err != nil {
		log.Fatalf("error reading authcode %s", err)
	}
	log.Printf("authcode: %s", authCode)

	time.Sleep(DELAY * time.Second)
	token, err := redditOAuthConfig.Exchange(ctx, authCode)
	if err != nil {
		panic(err)
	}
	log.Printf("token: %s", token)
	client := getRedditClient(ctx, token)
	log.Printf("client: %s", client)

	time.Sleep(DELAY * time.Second)

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
