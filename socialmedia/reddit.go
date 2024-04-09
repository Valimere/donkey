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
	Debug            bool
}

type redditResponse struct {
	Data struct {
		After    string `json:"after"`
		Before   string `json:"before"`
		Children []struct {
			Data struct {
				ID          string  `json:"id"`
				Title       string  `json:"title"`
				SelfText    string  `json:"selftext"`
				Author      string  `json:"author"`
				NumComments int     `json:"num_comments"`
				UpVotes     int     `json:"ups"`
				CreatedUTC  float64 `json:"created_utc"`
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
	Debug     bool
}

func (d *dumpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add User-Agent Header to request
	req.Header.Add("User-Agent", d.transport.UserAgent)

	if d.Debug {
		dump, err := httputil.DumpRequestOut(req, false)
		if err != nil {
			return nil, err
		}
		log.Printf("HTTP Request:\n%s\n", dump)
	}
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

func NewClient(debugFlag bool) *Client {
	httpClient := &http.Client{
		Transport: &dumpTransport{
			transport: &Transport{
				UserAgent: userAgent,
			},
			Debug: debugFlag,
		},
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	return &Client{
		OAuthConfig: getOAuthConfig(),
		HttpClient:  httpClient,
		Port:        8080,
		Throttle:    time.Tick(time.Second),
		Context:     ctx,
		Debug:       debugFlag,
	}
}

func NewClientWithToken(token *oauth2.Token, debugFlag bool) *Client {
	httpClient := &http.Client{
		Transport: &dumpTransport{
			transport: &Transport{
				UserAgent: userAgent,
			},
			Debug: debugFlag,
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
		Debug:       debugFlag,
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

func processRedditResponse(resp *http.Response) (RedditResponse, error) {
	var jsonData redditResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return RedditResponse{}, err
	}
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
		createdTime := time.Unix(int64(child.Data.CreatedUTC), 0).UTC()
		rr.Posts = append(rr.Posts, Post{
			ID:          child.Data.ID,
			Title:       child.Data.Title,
			Body:        child.Data.SelfText,
			Author:      child.Data.Author,
			NumComments: child.Data.NumComments,
			UpVotes:     child.Data.UpVotes,
			Created:     createdTime,
		})
	}
	return rr, nil
}

type PaginationOptions struct {
	Before string
	After  string
}

// FetchPosts retrieves the latest posts from a subreddit using the Reddit API.
// It makes a GET request to the subreddit's "new" endpoint and returns a RedditResponse object containing the posts.
// The method utilizes the client's Throttle channel to throttle requests.
// The method requires the subreddit name as the first argument and supports optional PaginationOptions.
// If provided, PaginationOptions determine the "before" and "after" query parameters in the request URL.
// The method sets the "Accept" and "Authorization" headers in the request and handles any errors that occur during the HTTP request.
// It also logs information about the rate limit headers received in the HTTP response.
// The method returns the RedditResponse and an error if one occurs.
// Example usage:
//
//	client := &Client{}
//	resp, err := client.FetchPosts(context.Background(), "golang")
func (c *Client) FetchPosts(ctx context.Context, subreddit string, opts ...PaginationOptions) (RedditResponse, error) {
	// Throttling requests
	<-c.Throttle
	baseURL := fmt.Sprintf("https://oauth.reddit.com/r/%s/new.json", subreddit)
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		return RedditResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token.AccessToken)
	if len(opts) > 0 {
		params := url.Values{}
		if opts[0].Before != "" {
			params.Add("before", opts[0].Before)
		}
		if opts[0].After != "" {
			params.Add("after", opts[0].After)
		}
	}
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return RedditResponse{}, err
	}

	// Inspect rate limit headers right after the HTTP request is made
	log.Printf("Ratelimit-Used: %s, Ratelimit-Remaining: %s, Ratelimit-Reset: %s\n",
		resp.Header.Get("X-Ratelimit-Used"),
		resp.Header.Get("X-Ratelimit-Remaining"),
		resp.Header.Get("X-Ratelimit-Reset"))
	defer resp.Body.Close()
	return processRedditResponse(resp)
}

// callbackHandler handles the callback request from the OAuth server.
// It extracts the authorization code from the request URL, stores it in the Client's AuthCode field,
// and responds to the request with a message indicating that the authorization code has been received.
// If there is an error while writing the response, an HTTP 500 error is returned.
// The method is expected to be used in conjunction with the StartServer method.
// Example usage:
//
//	c := &Client{}
//	c.StartServer(context.Background())
//	// User completes OAuth authorization flow and receives an authorization code
//	// The authorization code is automatically stored in c.AuthCode by the callbackHandler method
//	token, err := c.ExchangeAuthCode(context.Background())
func (c *Client) callbackHandler(w http.ResponseWriter, r *http.Request) {
	c.AuthCode = r.URL.Query().Get("code")
	log.Printf("code received: %s", c.AuthCode)
	_, err := fmt.Fprintf(w, "Authorization code received. You can close this window.")
	if err != nil {
		http.Error(w, "Error occurred while writing response in callback:", http.StatusInternalServerError)
		log.Fatalf("Error when writing response in callback: %v", err)
	}
}

// ExchangeAuthCode exchanges the authorization code for an OAuth2 token.
// It waits for the Throttle channel to receive a signal before proceeding,
// ensuring that the rate limit is respected.
// It uses the OAuthConfig to make the token exchange request.
// If successful, it stores the token in the Client's Token field and returns it.
// If there is an error, it returns nil for the token and the error.
//
// Example usage:
//
//	c := &Client{}
//	c.StartServer(context.Background())
//	// User completes OAuth authorization flow and receives an authorization code
//	// The authorization code is automatically stored in c.AuthCode by the callbackHandler method
//	token, err := c.ExchangeAuthCode(context.Background())
func (c *Client) ExchangeAuthCode(ctx context.Context) (*oauth2.Token, error) {
	<-c.Throttle
	t, err := c.OAuthConfig.Exchange(ctx, c.AuthCode)
	if err != nil {
		return nil, err
	}
	c.Token = t
	return t, nil
}
