package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Valimere/donkey/socialmedia"
	"log"
	"os"
	"strings"
	"time"
)

var (
	DELAY = time.Duration(1)
)

func main() {
	subredditsArg := flag.String("r", "music", "comma-separated list of subreddits i.e. \"funny, music\"")
	logFile := flag.String("log", "", "path to log file (optional)")
	flag.Parse()

	// Configure logging
	if *logFile != "" {
		file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Error opening log file: %v", err)
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				log.Fatalf("error trying to close the file %v", err)
			}
		}(file)
		log.SetOutput(file)
	} else {
		log.SetOutput(os.Stdout)
	}

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
	client := socialmedia.NewClient()
	ctx := context.Background()

	// Start the HTTP server in a goroutine and get the authorization URL
	serverErr := client.StartServer(ctx)

	if serverErr != nil {
		log.Fatalf("Error in server: %s", serverErr)
	}

	token, err := client.ExchangeAuthCode(ctx)

	log.Printf("token received: %s", token)

	if err != nil {
		panic(err)
	}

	for _, subreddit := range subreddits {
		posts, err := client.FetchPosts(ctx, subreddit)
		if err != nil {
			panic(err)
		}
		for _, post := range posts {
			fmt.Printf("Post ID: %s\n", post.ID)
			fmt.Printf("Title: %s\n", post.Title)
			fmt.Printf("Body: %s\n", post.Body)
			fmt.Printf("Author: %s\n", post.Author)
			fmt.Println()
		}
	}

	// For demonstration, printing JSON response
	//var jsonData map[string]interface{}
	//err = json.Unmarshal(posts, &jsonData)
	//if err != nil {
	//	log.Printf("unable to unmarshall json error: %v", err)
	//}
	//fmt.Println(jsonData)
}
