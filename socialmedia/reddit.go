package socialmedia

import (
	"context"
	"encoding/json"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"fmt"
	"golang.org/x/oauth2"
	"io"
	"log"
	"net/http"
)

const (
	redirectURL = "http://localhost:8080/callback"
	authURL     = "https://www.reddit.com/api/v1/authorize"
	tokenURL    = "https://www.reddit.com/api/v1/access_token"
	authScope   = "read"
)

var (
	clientID     = os.Getenv("REDDIT_CLIENT_ID")
	clientSecret = os.Getenv("REDDIT_SECRET")
	userAgent    = os.Getenv("REDDIT_USER_AGENT")
)

type Client struct {
	OAuthConfig      *oauth2.Config
	AuthorizationURL string
	AuthCode         string
	ServerErr        error
	Token            *oauth2.Token
	Port             int
	Throttle         <-chan time.Time
	HttpClient       *http.Client
	Context          context.Context
}

type redditResponse struct {
	Data struct {
		After    string `json:"after"`
		Before   string `json:"before"`
		Children []struct {
			Data struct {
				ID          string `json:"id"`
				Title       string `json:"title"`
				SelfText    string `json:"selftext"`
				Author      string `json:"author"`
				NumComments int    `json:"num_comments"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type Transport struct {
	UserAgent string
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return http.DefaultTransport.RoundTrip(req)
}

type dumpTransport struct {
	transport *Transport
}

func (d *dumpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add User-Agent Header to request
	req.Header.Add("User-Agent", d.transport.UserAgent)

	dump, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		return nil, err
	}
	log.Printf("HTTP Request:\n%s\n", dump)
	return d.transport.RoundTrip(req)
}

func getOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{authScope},
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}
}

func NewClient() *Client {
	httpClient := &http.Client{
		Transport: &dumpTransport{
			transport: &Transport{
				UserAgent: userAgent,
			},
		},
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	return &Client{
		OAuthConfig: getOAuthConfig(),
		HttpClient:  httpClient,
		Port:        8080,
		Throttle:    time.Tick(time.Second),
		Context:     ctx,
	}
}

func NewClientWithToken(token *oauth2.Token) *Client {
	httpClient := &http.Client{
		Transport: &dumpTransport{
			transport: &Transport{
				UserAgent: userAgent,
			},
		},
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	return &Client{
		OAuthConfig: getOAuthConfig(),
		Token:       token,
		HttpClient:  httpClient,
		Port:        8080,
		Throttle:    time.Tick(time.Second),
		Context:     ctx,
	}
}

func (c *Client) StartServer(ctx context.Context) error {
	log.Printf("Starting http server on port %d", c.Port)
	c.AuthorizationURL = c.OAuthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)

	http.HandleFunc("/callback", c.callbackHandler)
	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil))
	}()

	fmt.Printf("Go to the following link in your browser:\n%s\n", c.AuthorizationURL)
	for c.AuthCode == "" && c.ServerErr == nil {
		time.Sleep(100 * time.Millisecond)
	}
	return c.ServerErr
}

func (c *Client) FetchPosts(ctx context.Context, subreddit string) (RedditResponse, error) {
	<-c.Throttle
	baseURL := fmt.Sprintf("https://oauth.reddit.com/r/%s/new.json", subreddit)
	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return RedditResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return RedditResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return RedditResponse{}, err
	}
	var jsonData redditResponse
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		log.Printf("\nUnparsable: \n%s\n", body)
		return RedditResponse{}, err
	}
	rr := RedditResponse{
		Before: jsonData.Data.Before,
		After:  jsonData.Data.After,
	}
	for _, child := range jsonData.Data.Children {
		rr.Posts = append(rr.Posts, Post{
			ID:          child.Data.ID,
			Title:       child.Data.Title,
			Body:        child.Data.SelfText,
			Author:      child.Data.Author,
			NumComments: child.Data.NumComments,
		})
	}
	return rr, nil
}

func (c *Client) FetchPostsBA(ctx context.Context, subreddit string, before string, after string) (RedditResponse, error) {
	<-c.Throttle
	baseURL, err := url.Parse(fmt.Sprintf("https://oauth.reddit.com/r/%s/new.json", subreddit))
	if err != nil {
		return RedditResponse{}, err
	}

	// Prepare Query Parameters
	params := url.Values{}
	if before != "" {
		params.Add("before", before)
	}
	if after != "" {
		params.Add("after", after)
	}
	baseURL.RawQuery = params.Encode() // Encode URL parameters

	req, err := http.NewRequest("GET", baseURL.String(), nil)
	if err != nil {
		return RedditResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return RedditResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return RedditResponse{}, err
	}
	var jsonData redditResponse
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		log.Printf("\nUnparsable: \n%s\n", body)
		return RedditResponse{}, err
	}
	rr := RedditResponse{
		Before: jsonData.Data.Before,
		After:  jsonData.Data.After,
	}
	for _, child := range jsonData.Data.Children {
		rr.Posts = append(rr.Posts, Post{
			ID:          child.Data.ID,
			Title:       child.Data.Title,
			Body:        child.Data.SelfText,
			Author:      child.Data.Author,
			NumComments: child.Data.NumComments,
		})
	}
	return rr, nil
}

func (c *Client) callbackHandler(w http.ResponseWriter, r *http.Request) {
	c.AuthCode = r.URL.Query().Get("code")
	log.Printf("code received: %s", c.AuthCode)
	_, err := fmt.Fprintf(w, "Authorization code received. You can close this window.")
	if err != nil {
		http.Error(w, "Error occurred while writing response in callback:", http.StatusInternalServerError)
		log.Fatalf("Error when writing response in callback: %v", err)
	}
}

func (c *Client) ExchangeAuthCode(ctx context.Context) (*oauth2.Token, error) {
	<-c.Throttle
	t, err := c.OAuthConfig.Exchange(ctx, c.AuthCode)
	if err != nil {
		return nil, err
	}
	c.Token = t
	return t, nil
}
