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

// handleFatalErrors is a helper function to make error handling more uniform
func handleFatalErrors(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

// configureLogOutput : set the log output destination
func configureLogOutput(filepath *string) {
	if *filepath != "" {
		file, err := os.OpenFile(*filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
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
func fetchAndPrint(client *socialmedia.Client, subreddit string) {
	var after string
	for {
		resp, err := client.FetchPosts(context.Background(), subreddit, socialmedia.PaginationOptions{After: after})
		handleFatalErrors(err, fmt.Sprintf("Error fetching posts for subreddit: %s", subreddit))
		for _, post := range resp.Posts {
			err := statistics.SaveUniquePost(post.ID, post.Author, post.Title, post.Upvotes, post.NumComments)
			if err != nil {
				log.Printf("Failed to save post statistic error:%s\n", err)
			}
			if client.Debug {
				fmt.Printf("Post ID: %s, NumComments:%4d ,Author:%24s, Title: %s\n",
					post.ID, post.NumComments, post.Author, post.Title)
			}
		}
		after = resp.After
	}
}

func printStatisticsAndExit() {
	authorStatistics, err := statistics.GetTopPoster()

	if err != nil {
		fmt.Printf("Error getting author statistics: %s\n", err)
		os.Exit(1)
		return
	}

	fmt.Printf("\n\nAuthor Statistics:\n")
	for _, authorStatistic := range *authorStatistics {
		fmt.Printf("Author: %s, PostsCount: %d\n", authorStatistic.Author, authorStatistic.TotalPosts)
	}
	postStatistics, err := statistics.GetTopPosts()
	if err != nil {
		fmt.Printf("Error in getting post statistics %s\n", err)
		os.Exit(1)
		return
	}
	for _, postStatistic := range *postStatistics {
		fmt.Printf("\n\nPost Statistics:\n")
		fmt.Printf("Post ID: %s, Author: %s, UpVotes: %d, Comments: %d\n", postStatistic.PostID, postStatistic.Author, postStatistic.UpVotes, postStatistic.NumComments)
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
	debugFlag := flag.Bool("debug", false, "enable debug mode")
	flag.Parse()

	// Configure logging
	configureLogOutput(logFile)

	// Create a channel to listen for OS signals, print statistics on ctl + c
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		printStatisticsAndExit()
	}()

	subreddits := parseSubreddits(subredditsArg)

	log.Printf("Subreddits chosen: %v\n", subreddits)

	// Initialize db connection and create store
	db.InitDB(*debugFlag)
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
		// If the token exists, and it has not expired, use it
		client = socialmedia.NewClientWithToken(dbToken, *debugFlag)
	} else {
		client = socialmedia.NewClient(*debugFlag)

		serverErr := client.StartServer(context.Background())
		handleFatalErrors(serverErr, "Error in server")

		token, err := client.ExchangeAuthCode(context.Background())
		handleFatalErrors(err, "Failed to exchange auth code")

		log.Printf("token received: %s", token.AccessToken)

		err = store.SaveToken(token)
		handleFatalErrors(err, "Failed to save the token")
	}

	fetchAndPrint(client, subreddits[0])
}
