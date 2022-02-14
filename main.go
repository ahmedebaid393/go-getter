package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	downloader "github.com/ahmedebaid393/go-getter/downloader"
)

func main() {
	startTime := time.Now()

	// get cli flags
	url := flag.String("url", "", "Downloadable URL of file")
	targetPath := flag.String("destination", "./downloads", "Downloads destination path")
	totalSections := flag.Int("sections_num", 10, "Number of download sections")

	// download provided URL file
	err := downloader.Do(*url, *targetPath, *totalSections)
	if err != nil {
		log.Fatalf("An error occurred. Error says: %s\n", err)
	}

	downloadTime := time.Now().Sub(startTime).Seconds()
	fmt.Printf("Download completed in %v seconds\n", downloadTime)
}
