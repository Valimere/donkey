package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Valimere/donkey/db"
	"github.com/Valimere/donkey/socialmedia"
	"github.com/Valimere/donkey/statistics"
	"github.com/Valimere/donkey/store"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// handleFatalErrors is a helper function to make error handling more uniform
func handleFatalErrors(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
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
	log.Printf("Subreddits chosen: %v\n", subreddits)
	return subreddits
}

// Fetch and print posts from a single subreddit
func fetchAndPrint(client *socialmedia.Client, subreddits []string, dbStore store.Store) {
	var wg sync.WaitGroup

	for _, subreddit := range subreddits {
		wg.Add(1)
		go func(subreddit string) {
			defer wg.Done()
			var after string
			for {
				resp, err := client.FetchPosts(context.Background(), subreddit, socialmedia.PaginationOptions{After: after})
				handleFatalErrors(err, fmt.Sprintf("Error fetching posts for subreddit: %s", subreddit))
				for _, post := range resp.Posts {
					if post.Created.After(client.ProgramStartTime) {
						err := statistics.SaveUniquePost(dbStore, &post)
						if err != nil {
							log.Printf("Failed to save post statistic error:%s\n", err)
						}
						if client.Debug {
							fmt.Printf("Post PostID: %s, NumComments:%4d, Subreddit: %12s, Author:%24s, Title: %s\n",
								post.PostID, post.NumComments, post.SubReddit, post.Author, post.Title)
						}
					}

				}
				after = resp.After
			}
		}(subreddit)
	}
	wg.Wait()

}

func printStatisticsAndExit(dbStore store.Store) {
	authorStatistics, err := statistics.GetTopPoster(dbStore)
	if err != nil {
		fmt.Printf("Error getting author statistics: %s\n", err)
		os.Exit(1)
		return
	}

	fmt.Printf("\n\nAuthor Statistics:\n")
	for _, authorStatistic := range authorStatistics {
		fmt.Printf("Author: %s, PostsCount: %d\n", authorStatistic.Author, authorStatistic.TotalPosts)
	}
	postStatistics, err := statistics.GetTopPosts(dbStore)
	if err != nil {
		fmt.Printf("Error in getting post statistics %s\n", err)
		os.Exit(1)
		return
	}
	fmt.Printf("\n\nPost Statistics:\n")
	for _, postStatistic := range postStatistics {
		fmt.Printf("Post PostID: %8s, UpVotes: %4d, Comments: %4d, Author: %24s\n",
			postStatistic.PostID, postStatistic.UpVotes, postStatistic.NumComments, postStatistic.Author)
	}

	os.Exit(0)
}

func clearStatistics(dbStore store.Store) {
	// Clear all rows in the AuthorStatistic table.
	err := dbStore.ClearAuthorStatistics()
	if err != nil {
		// Log the error
		log.Println("Error clearing author_statistics:", err)
	}
	err = dbStore.ClearPosts()
	if err != nil {
		log.Println("error clearing posts:", err)
	}
}

func main() {
	subredditsArg := flag.String("r", "Askreddit", "comma-separated list of subreddits i.e. \"Askreddit, music\"")
	debugFlag := flag.Bool("debug", false, "enable debug mode")
	flag.Parse()

	// Initialize db connection and create store
	dbInstance, err := db.InitDB(*debugFlag)
	if err != nil {
		handleFatalErrors(err, "Failed to initialize database: ")
	}

	var dbStore store.Store = &db.DbStore{DB: dbInstance}
	clearStatistics(dbStore)

	// Check if a token exists in the database
	dbToken, err := dbStore.GetToken()
	if err != nil {
		if err.Error() == "record not found" {
			log.Println("No existing token found in the database. Requesting a new one.")
		} else {
			log.Fatalf("Unexpected error retrieving token from the store: %v\n", err)
		}
	}

	// Create a channel to listen for OS signals, print statistics on ctl + c
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		printStatisticsAndExit(dbStore)
	}()

	subreddits := parseSubreddits(subredditsArg)

	var smClient *socialmedia.Client

	if dbToken.Valid() && !dbToken.Expiry.Before(time.Now()) {
		// If the token exists, and it has not expired, use it
		smClient = socialmedia.NewClientWithToken(dbToken, *debugFlag)
	} else {
		smClient = socialmedia.NewClient(*debugFlag)

		serverErr := smClient.StartServer(context.Background())
		handleFatalErrors(serverErr, "Error in server")

		token, err := smClient.ExchangeAuthCode(context.Background())
		handleFatalErrors(err, "Failed to exchange auth code")

		log.Printf("token received: %s", token.AccessToken)

		err = dbStore.SaveToken(token)
		handleFatalErrors(err, "Failed to save the token")
	}

	fetchAndPrint(smClient, subreddits, dbStore)
}
