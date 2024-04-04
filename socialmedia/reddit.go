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

const (
	redirectURL        = "http://localhost:8080/callback"
	authURL            = "https://www.reddit.com/api/v1/authorize"
	tokenURL           = "https://www.reddit.com/api/v1/access_token"
	authorizationScope = "read"
)

var (
	clientID                 = os.Getenv("REDDIT_CLIENT_ID")
	clientSecret             = os.Getenv("REDDIT_SECRET")
	_            SocialMedia = &Client{} // Ensure reddit.Client implements SocialMedia interface at compile time.
)

// helper function to build oauth2 config
func getOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{authorizationScope},
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}
}

type Client struct {
	OAuthConfig      *oauth2.Config
	AuthorizationURL string
	AuthCode         string
	ServerErr        error
	Token            *oauth2.Token
	Port             int              // Add Port field
	Throttle         <-chan time.Time // Add Throttle field. This is a receive-only channel
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

// NewClient You can create a constructor function for reddit.Client which initializes OAuthConfig
func NewClient() *Client {
	return &Client{
		OAuthConfig: getOAuthConfig(),
		Port:        8080,                   // Default port
		Throttle:    time.Tick(time.Second), // Default rate is 1 request per second
	}
}

func NewClientWithToken(token *oauth2.Token) *Client {
	return &Client{
		OAuthConfig: getOAuthConfig(),
		Token:       token,
		Port:        8080,
		Throttle:    time.Tick(time.Second),
	}
}

func (c *Client) StartServer(ctx context.Context) error {
	log.Printf("Starting http server on port %d", c.Port) // Use the port from the client
	c.AuthorizationURL = c.OAuthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)

	http.HandleFunc("/callback", c.callbackHandler)
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil); err != nil { // Use the port from the client
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

func (c *Client) FetchPosts(ctx context.Context, subreddit string) ([]Post, error) {

	<-c.Throttle // rate limit our posts respecting the throttle
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

func (c *Client) callbackHandler(w http.ResponseWriter, r *http.Request) {
	c.AuthCode = r.URL.Query().Get("code")
	log.Printf("code received: %s", c.AuthCode)
	_, err := fmt.Fprintf(w, "Authorization code received. You can close this window.")
	if err != nil {
		log.Println("Error when writing response in callback:", err)
	}
}

func (c *Client) ExchangeAuthCode(ctx context.Context) (*oauth2.Token, error) {
	<-c.Throttle // limit our requests based on throttle
	t, err := c.OAuthConfig.Exchange(ctx, c.AuthCode)
	if err != nil {
		return nil, err
	}

	c.Token = t
	return t, nil
}
