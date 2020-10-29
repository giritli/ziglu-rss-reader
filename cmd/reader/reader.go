package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"reader/internal/api"
	"reader/internal/feed"
	"reader/internal/reader"
	"reader/internal/storage"
	"sync"
	"time"
)

func main() {

	// Define file flag which needs to point to a JSON file containing
	// an array of feed URL's/
	var feedFile = flag.String("file", "", "file of feeds")
	flag.Parse()

	if *feedFile == "" {
		fmt.Println("Please specify -file argument")
		os.Exit(1)
	}

	f, err := os.Open(*feedFile)
	if err != nil {
		fmt.Printf("Could not open feed file: %v\n", err)
		os.Exit(1)
	}

	var feedLinks []string
	if err := json.NewDecoder(f).Decode(&feedLinks); err != nil {
		fmt.Printf("Could not parse feed file as JSON: %v\n", err)
		os.Exit(1)
	}

	ctx, cf := context.WithCancel(context.Background())
	go catchSignal(cf)

	var s storage.Storage = storage.NewInMemoryStorage(30)

	// Loop through all given feeds and try to store them for later retrieval
	for _, fl := range feedLinks {
		u, err := url.Parse(fl)
		if err != nil {
			log.Printf("Could not parse feed link as URL: %v\n", err)
			continue
		}

		err = s.Store(&feed.Feed{
			FeedLink:   u,
			ModifiedAt: time.Time{},
		}, nil)
		if err != nil {
			log.Printf("Could not store feed URL: %v\n", err)
			continue
		}
	}

	r := reader.NewReader(s)

	feeds, err := s.Feeds()
	if err != nil {
		log.Printf("Could not retrieve available feeds from storage: %v\n", err)
		os.Exit(1)
	}

	errChan := r.Update(ctx, feeds)
	go func() {
		for err := range errChan {
			fmt.Printf("Error updating feed: %v\n", err)
		}
	}()

	// Initialise our web server
	srv := http.Server{
		Addr:    ":8080",
		Handler: api.NewAPI(s),
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		// If the server errors out with anything other than server close,
		// panic the error.
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Error returned from server: %v\n", err)
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("Shutting down the server, waiting for remaining requests")

		// Create a context which times out after 1 minute. This should give
		// ample time for all remaining requests to be processed before
		// force shutting the server.
		timeoutCtx, _ := context.WithTimeout(context.Background(), 1*time.Minute)
		if err := srv.Shutdown(timeoutCtx); err != nil {
			log.Fatalf("Error shutting down server: %v\n", err)
		}
	}

	wg.Wait()
}

// catchSignal will cancel given context if a termination signal is passed to the program
func catchSignal(cancelFunc context.CancelFunc) {
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	<-s
	cancelFunc()
}
