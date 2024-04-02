package main

import (
	"flag"
	"log"
	"os"
	"strings"
)

func main() {
	subredditsArg := flag.String("r", "music", "comma-separated list of subreddits i.e. \"funny, music\"")
	logFile := flag.String("log", "", "path to log file (optional)")
	flag.Parse()

	// Configure logging
	if *logFile != "" {
		// creating a file if it doesn't exist, for write only, and appends, perms: readable and writable by all users
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

	log.Printf("\nStarting app, Subreddits chosen: %v\n", subreddits)
}
