package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/sync/semaphore"
)

// BadURL describe a link that cannot be scanned
type BadURL struct {
	StatusCode int
	URL        string
}

//go:embed sitemap_template
var sitemapTemplate string

// To maximeize throughput of the scan, we setup a semaphore
// The semaphore automaticcaly limit the number of go routines
// to execute
var lock *semaphore.Weighted

// The context instead trace the pool of go routines
var ctx = context.TODO()

// Default paramters used for scanning
// The maxConnections describe the number of connection that can be open
// in the same time
var maxConnections = 120

// The system cannot determinate when all URL all scanned,
// For avoid infinite loop problem, the system check the latest
// activity. If there isn't activities after the number of SECONDs provided
// below, the infinite loop finish.
// NOTE:
// Consider to change via CLI flags this param for sites with poor response
// or if you have a poor connection
var hearthBeatInterval = 3

// The last activity monitored
var lastActivityAt = time.Now()

// The count that monitoring the number
// of connections currently opened
var connectionsOpened = 0

// When a URL cannot be scanned, because not exist or because timeout
// or another reasons, it will store in this array
var linksFailed = []BadURL{}

// Instead if a URL is valid, will store here
var linksSuccessed = []string{}

// In any case all links are contained here
var allLinks = []string{}

// The domain extracted from the URL provided as first
// argument
var domain = ""

// The protocol extracted from the URL provided
// as first argument
var protocol = ""

// The main channel use to communicate
// the URLs to scan when found inside a page
var ch = make(chan string, 1)

// The waitgroup used to block the main loop until
// the heartbeat is stopped
var wg = new(sync.WaitGroup)
var rq = new(sync.WaitGroup)

func getPage(url string) (*goquery.Document, int, error) {
	response, err := http.Get(url)
	if err != nil {
		if strings.Contains(err.Error(), "too many open files") {
			return nil, 3, err
		}
		fmt.Println(err)
		return nil, 1, err

	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, response.StatusCode, errors.New("Not a valid page")
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, 2, err
	}

	return doc, 0, nil
}

func scanPage(page *goquery.Document) {
	page.Find("a").Each(func(i int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if ok == false {
			return
		}

		if isAllowedURL(href) == false {
			return
		}

		href = decorateURL(href)

		if isURLJustFound(&allLinks, href) == true {
			return
		}

		allLinks = append(allLinks, href)

		go func() {
			ch <- href
		}()

	})
}

func setLastActivity() {
	lastActivityAt = time.Now()
}

func decreaseConnectionsOpened() {
	connectionsOpened = connectionsOpened - 1
}

func increaseConnectionsOpened() {
	connectionsOpened = connectionsOpened + 1
}

func appendBadURL(badURL string, errorCode int) {
	linksFailed = append(linksFailed, BadURL{
		URL:        badURL,
		StatusCode: errorCode,
	})
}

func start(url string) {
	defer setLastActivity()

	page, statusCode, err := getPage(url)

	if err != nil && statusCode != 3 {
		appendBadURL(url, statusCode)
		return
	}

	// Max connections created.
	// try to repeat
	if statusCode == 3 {
		go func() {
			time.Sleep(time.Second)
			ch <- url
		}()
		return
	}

	if isURLJustFound(&linksSuccessed, url) == true {
		return
	}

	linksSuccessed = append(linksSuccessed, url)
	go scanPage(page)
}

func checkLifeActivity() {
	for {
		showScanStatus()
		now := time.Now()
		sub := now.Sub(lastActivityAt)
		if sub > (time.Duration(hearthBeatInterval) * time.Second) {
			wg.Done()
		}
		time.Sleep(time.Second)
	}
}

func waitForURLToScan() {
	for urlToScan := range ch {
		lock.Acquire(ctx, 1)

		// Eexecute the page fetching inside a anon function
		// In this case we can take advantage of defer logics
		go func(url string) {
			defer decreaseConnectionsOpened()
			defer lock.Release(1)
			increaseConnectionsOpened()
			start(url)
		}(urlToScan)
	}
}

func main() {
	// Defer somrthing
	defer close(ch)

	// 1. Start scanning page from endpoint provided
	// 2. Find all links (exclude duplicated ones)
	// 3. For each link found go to page and start scanning again
	// 4. The script finish when there are no more activities of scan
	// NOTE
	// You can follow only links in the same domain (no third levels, no external, ...)
	args := os.Args
	if len(args) <= 1 {
		panic("Missing URL argument")
	}

	url := args[len(args)-1]

	flagHeartBeatInterval := flag.Int("hb", hearthBeatInterval, "The heartbeat interval")
	flagMaxConnections := flag.Int("mc", maxConnections, "The allowed max connections")
	flag.Parse()

	hearthBeatInterval = *flagHeartBeatInterval
	maxConnections = *flagMaxConnections

	protocol = extractProtocol(url)
	domain = extractDomain(url)

	// Init lock
	lock = semaphore.NewWeighted(int64(maxConnections))

	wg.Add(1)

	go waitForURLToScan()
	go checkLifeActivity()

	// Scan the url provided
	ch <- url

	wg.Wait()

	showScanStatus()
	generateSitemap()

	for _, val := range linksFailed {
		fmt.Println(val.StatusCode, val.URL)
	}

}
