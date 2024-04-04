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
	REDDIT_CLIENT_ID = os.Getenv("REDDIT_CLIENT_ID")
	REDDIT_SECRET    = os.Getenv("REDDIT_SECRET")
	DELAY            = time.Duration(1)
	authCode         string
	serverErr        error
)

var redditOAuthConfig = &oauth2.Config{
	ClientID:     REDDIT_CLIENT_ID,
	ClientSecret: REDDIT_SECRET,
	RedirectURL:  "http://localhost:8080/callback",
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

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	authCode = r.URL.Query().Get("code")
	_, serverErr := fmt.Fprintf(w, "Authorization code received. You can close this window now.")
	if serverErr != nil {
		log.Fatalf("authorization code received: %s", serverErr)
	}
}

func startServer() {
	http.HandleFunc("/callback", callbackHandler)
	http.ListenAndServe(":8080", nil)
}

func main() {
	ctx := context.Background()
	// Start the HTTP server in a goroutine
	go startServer()

	// Redirect the user to the authorization URL
	authURL := redditOAuthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser:\n%s\n", authURL)

	// Wait for the authorization code
	for authCode == "" && serverErr == nil {
		time.Sleep(2 * time.Second)
	}

	if serverErr != nil {
		log.Fatalf("Error in server: %s", serverErr)
	}

	token, err := redditOAuthConfig.Exchange(ctx, authCode)
	if err != nil {
		panic(err)
	}
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
