package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Valimere/donkey/db"
	"github.com/Valimere/donkey/socialmedia"
	"github.com/Valimere/donkey/statistics"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// handleFatalErrors is a helper function to make your error handling more uniform
func handleFatalErrors(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

// configureLogOutput : set the log output destination
func configureLogOutput(logFile *string) {
	if *logFile != "" {
		file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		handleFatalErrors(err, "Error opening log file")
		defer file.Close()
		log.SetOutput(file)
	} else {
		log.SetOutput(os.Stdout)
	}
}

// parseSubreddits : Clean up the subreddits input
func parseSubreddits(subredditsArg *string) []string {
	// Split the subredditsArg into a slice of uncleanSubreddits
	uncleanSubreddits := strings.Split(*subredditsArg, ",")

	// Create a new slice for the cleaned subreddits
	var subreddits []string
	for _, subreddit := range uncleanSubreddits {
		subreddit = strings.TrimSpace(subreddit)
		if subreddit != "" {
			subreddits = append(subreddits, subreddit)
		}
	}
	return subreddits
}

// Fetch and print posts from a single subreddit
func fetchAndPrint(client *socialmedia.Client, subreddit string) (socialmedia.RedditResponse, error) {
	resp, err := client.FetchPosts(context.Background(), subreddit)
	handleFatalErrors(err, fmt.Sprintf("Error fetching posts for subreddit: %s", subreddit))
	fmt.Printf("After: %s\n", resp.After)
	for _, post := range resp.Posts {
		err := statistics.SaveUniquePost(post.ID, post.Author)
		if err != nil {
			log.Printf("Failed to save post statistic error:%s\n", err)
		}
		fmt.Printf("Post ID: %s, NumComments:%4d ,Author:%24s, Title: %s\n",
			post.ID, post.NumComments, post.Author, post.Title)
	}
	return resp, nil
}

// Fetch and print posts continuously from a single subreddit
func continuousFetchAndPrint(client *socialmedia.Client, subreddit string, before string, after string) {
	for {
		// Sleep for a specified duration before the next iteration
		<-client.Throttle
		resp, err := client.FetchPostsBA(context.Background(), subreddit, before, after)
		handleFatalErrors(err, fmt.Sprintf("Error fetching posts for subreddit: %s", subreddit))
		fmt.Printf("Before: %s, After: %s\n", resp.Before, resp.After)
		for _, post := range resp.Posts {
			err := statistics.SaveUniquePost(post.ID, post.Author)
			if err != nil {
				log.Printf("Failed to save post statistic error:%s\n", err)
			}
			fmt.Printf("Post ID: %s, NumComments:%4d ,Author:%24s, Title: %s\n",
				post.ID, post.NumComments, post.Author, post.Title)
		}
		before = resp.Before
		after = resp.After
	}
}

func printAuthorStatisticsAndExit() {
	authorStatistics, err := statistics.GetTopPoster()

	if err != nil {
		fmt.Printf("Error getting author statistics: %s", err)
		os.Exit(1)
		return
	}

	fmt.Printf("\n\nAuthor Statistics:\n")
	for _, authorStatistic := range *authorStatistics {
		fmt.Printf("Author: %s, PostsCount: %d\n", authorStatistic.Author, authorStatistic.TotalPosts)
	}

	os.Exit(0)
}

func clearAuthorStatistics() {
	// Clear all rows in the AuthorStatistic table.
	err := db.DB.Exec("DELETE FROM author_statistics").Error
	if err != nil {
		// Log the error
		log.Println("Error clearing author_statistics:", err)
	}
	err = db.DB.Exec("DELETE FROM posts").Error
	if err != nil {
		log.Println("error clearing posts:", err)
	}
}

func main() {
	subredditsArg := flag.String("r", "music", "comma-separated list of subreddits i.e. \"funny, music\"")
	logFile := flag.String("log", "", "path to log file (optional)")
	flag.Parse()

	// Configure logging
	configureLogOutput(logFile)

	// Create a channel to listen for OS signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		printAuthorStatisticsAndExit()
	}()

	subreddits := parseSubreddits(subredditsArg)

	log.Printf("Subreddits chosen: %v\n", subreddits)

	// Initialize db connection and create store
	db.InitDB()
	store := &db.DBStore{DB: db.DB}
	clearAuthorStatistics()

	// Check if a token exists in the database
	dbToken, err := store.GetToken()
	if err != nil {
		if err.Error() == "record not found" {
			log.Println("No existing token found in the database. Requesting a new one.")
		} else {
			log.Fatalf("Unexpected error retrieving token from the store: %v\n", err)
		}
	}

	var client *socialmedia.Client

	if dbToken.Valid() && !dbToken.Expiry.Before(time.Now()) {
		// If the token exists and it has not expired, use it
		client = socialmedia.NewClientWithToken(dbToken)
	} else {
		client = socialmedia.NewClient()

		serverErr := client.StartServer(context.Background())
		handleFatalErrors(serverErr, "Error in server")

		token, err := client.ExchangeAuthCode(context.Background())
		handleFatalErrors(err, "Failed to exchange auth code")

		log.Printf("token received: %s", token.AccessToken)

		err = store.SaveToken(token)
		handleFatalErrors(err, "Failed to save the token")
	}

	var resp socialmedia.RedditResponse
	//for _, subreddit := range subreddits {
	//	resp, _ = fetchAndPrint(client, subreddit)
	//}
	resp, _ = fetchAndPrint(client, subreddits[0])
	continuousFetchAndPrint(client, subreddits[0], resp.Before, resp.After)
}
