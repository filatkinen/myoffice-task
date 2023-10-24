package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/filatkinen/myoffice-task/internal/urlquery"
)

var (
	maxThreads int
	fileURL    string
	userAgent  string
)

func main() {
	flag.Usage = func() {
		about()
		fmt.Println("Usage of this CLI utility:")
		flag.PrintDefaults()
	}
	flag.IntVar(&maxThreads, "m", 1000, "max threads run simultaneously")
	flag.StringVar(&fileURL, "f", "", "file name with URL")
	flag.StringVar(&fileURL, "u", "Go-http-client/1.1", "userAgent string")
	flag.Parse()

	if len(os.Args) == 1 || fileURL == "" {
		fmt.Println("error using: flag -f required")
		flag.Usage()
		return
	}

	inFile, err := os.Open(fileURL)
	if err != nil {
		log.Fatal("Error opening file:", err)
	}
	defer func() {
		if err = inFile.Close(); err != nil {
			log.Printf("Error closing file %v:", err)
		}
	}()

	query, err := urlquery.New(inFile, os.Stdout, maxThreads, userAgent)
	if err != nil {
		log.Printf("got error creating urlquery:%s", err)
		return
	}

	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGINT, syscall.SIGTERM)

	sigFinish := make(chan struct{})

	timeStart := time.Now()

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("Starting processing...")
		query.Start()
		close(sigFinish)
	}()

	select {
	case <-sigFinish:
		log.Println("Process is finished")
	case <-sigTerm:
		log.Println("Got termination signal. Exiting...")
		query.Stop()
	}

	wg.Wait()

	log.Println("Time to take:", time.Since(timeStart).Truncate(time.Millisecond))

	fmt.Println("\nResults of URL processing:")
	fmt.Println(query)
}

func about() {
	fmt.Println("This CLI reads URL from file and gets information from them: size, request time")
}
