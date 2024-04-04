package socialmedia

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Client struct {
	OAuthConfig      *oauth2.Config
	AuthorizationURL string
	AuthCode         string
	ServerErr        error
	Token            *oauth2.Token
}

type redditResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				SelfText string `json:"selftext"`
				Author   string `json:"author"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

// Ensure reddit.Client implements SocialMedia interface at compile time.
var _ SocialMedia = &Client{}
var PORT = 8080

// NewClient You can create a constructor function for reddit.Client which initializes OAuthConfig
func NewClient() *Client {
	return &Client{
		OAuthConfig: &oauth2.Config{
			ClientID:     os.Getenv("REDDIT_CLIENT_ID"),
			ClientSecret: os.Getenv("REDDIT_SECRET"),
			RedirectURL:  fmt.Sprintf("http://localhost:%d/callback", PORT),
			Scopes:       []string{"read"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.reddit.com/api/v1/authorize",
				TokenURL: "https://www.reddit.com/api/v1/access_token",
			},
		},
	}
}

func fetchSubredditPosts(ctx context.Context, client *http.Client, subreddit string) ([]byte, error) {
	resp, err := client.Get(fmt.Sprintf("https://oauth.reddit.com/r/%s/new", subreddit))
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("unable to close body")
		}
	}(resp.Body)

	return io.ReadAll(resp.Body)
}

func (c *Client) FetchPosts(ctx context.Context, subreddit string) ([]Post, error) {

	// Create an OAuth2 http.Client with your OAuthConfic
	httpClient := c.OAuthConfig.Client(ctx, c.Token)

	// Use the OAuth2 client to perform HTTP requests
	resp, err := httpClient.Get(fmt.Sprintf("https://oauth.reddit.com/r/%s/new", subreddit))

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jsonData redditResponse
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		return nil, err
	}

	var posts []Post

	for _, child := range jsonData.Data.Children {
		posts = append(posts, Post{
			ID:     child.Data.ID,
			Title:  child.Data.Title,
			Body:   child.Data.SelfText,
			Author: child.Data.Author,
		})
	}

	return posts, nil
}

func (c *Client) StartServer(ctx context.Context) error {
	log.Printf("Starting http server on port %d", PORT)
	// Start the HTTP server in a goroutine
	c.AuthorizationURL = c.OAuthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)

	http.HandleFunc("/callback", c.callbackHandler)
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", PORT), nil); err != nil {
			log.Fatalf("ListenAndServe Error: %+v", err)
		}
	}()

	// Wait for the authorization code
	fmt.Printf("Go to the following link in your browser:\n%s\n", c.AuthorizationURL)
	for c.AuthCode == "" && c.ServerErr == nil {
		time.Sleep(100 * time.Millisecond)
	}

	return c.ServerErr
}

func (c *Client) callbackHandler(w http.ResponseWriter, r *http.Request) {
	c.AuthCode = r.URL.Query().Get("code")
	_, c.ServerErr = fmt.Fprintf(w, "Authorization code received. You can close this window now.")
	if c.ServerErr != nil {
		log.Fatalf("authorization code received: %s", c.ServerErr)
	}
}

func (c *Client) ExchangeAuthCode(ctx context.Context) (*oauth2.Token, error) {
	t, err := c.OAuthConfig.Exchange(ctx, c.AuthCode)
	if err != nil {
		return nil, err
	}

	c.Token = t
	return t, nil
}
