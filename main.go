package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"quatermain/explorers"
	"quatermain/robots"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

// Page ...
type Page struct {
	Link          string
	CanonicalLink string
	StatusCode    int
}

const userAgent = "quatermain"

//go:embed help.txt
var helpTemplate string

//go:embed sitemap_template
var sitemapTemplate string

// To maximize the scan of the site, a semaphore for go routines will be created.
// It  will manage the maximum of simultaneous connections to the site
var lock *semaphore.Weighted
var ctx = context.TODO()

// The maxConnection parameter represents
// the maximum number of simultaneous connections that the semaphore will have to handle
var maxConnections = 120

// The requestInterval parameter represents the time interval between one connection to another.
// This parameter is very useful in limiting the number of requests over time.
// In the case of sites that use third-party services with a limited number of requests,
// it is important to set this parameter correctly
var requestInterval float64 = 0

// The connectionOpened variable represents the number of connections open to the website
var connectionsOpened = 0

// The domain extracted from the starting URL
var domain = ""

// The protocol extracted from the starting URL
var protocol = ""

// The main channel that takes care of collecting all requests for URLs to be scanned
var ch = make(chan string, 1)

// The waitgroup used to block the main loop
var wg = new(sync.WaitGroup)

// Robot instance to respect and check url based of robots.txt
var robot robots.IRobot

// The explorer
var explorer explorers.IExplorer

func explore(url string) {
	var canonicalURL string
	var statusCode int

	defer func() {
		time.Sleep(1 * time.Second)
		wg.Done()
	}()

	defer func() {
		if explorer.IsPageVisited(url) == true {
			return
		}
		explorer.AppendPage(&explorers.Page{
			Link:          url,
			CanonicalLink: canonicalURL,
			StatusCode:    statusCode,
		})
	}()

	if robot != nil && robot.CheckURL(url) == false {
		return
	}

	if explorer.IsPageVisited(url) == true {
		return
	}

	page, statusCode, err := explorer.Fetch(url)

	if err != nil {
		// Max connections created. try to repeat the request
		if strings.Contains(err.Error(), "too many open files") {
			go func() {
				time.Sleep(time.Second)
				ch <- url
			}()
			return
		}
		return
	}

	if explorer.BlockedByRobotsTag(page) == true {
		return
	}

	canonicalURL = explorer.FindCanonical(page)

	// Search for other links to scan
	go explorer.SearchLinks(page, func(link string) {
		ch <- link
	})
}

func waitForURLToScan() {
	for urlToScan := range ch {
		lock.Acquire(ctx, 1)
		wg.Add(1)

		time.Sleep(time.Duration(requestInterval) * time.Second)

		go func(url string) {
			defer decreaseConnectionsOpened()
			defer lock.Release(1)

			increaseConnectionsOpened()
			explore(url)
		}(urlToScan)
	}
}

// 1. Start scanning page from endpoint provided
// 2. Find all links (exclude duplicated ones)
// 3. For each link found go to page and start scanning again
// 4. The script finish when there are no more activities of scan
func main() {
	flagMaxConnections := flag.Int("c", maxConnections, "The allowed max connections.")
	flagRequestInterval := flag.Float64("i", requestInterval, "The interval to wait before a request")
	flagIsHelp := flag.Bool("h", false, "Print the help")
	flagIsUnethical := flag.Bool("u", true, "Force the explorer to be unethical and no respect crawler rules")
	flag.Parse()

	if *flagIsHelp == true {
		fmt.Println(helpTemplate)
		return
	}

	// Defer the closing of the channel
	defer close(ch)

	url := getURLFromArguments(os.Args)
	if url == "" {
		log.Fatal("Missing URL argument")
		return
	}
	protocol = extractProtocol(url)
	domain = extractDomain(url)

	maxConnections = *flagMaxConnections
	if maxConnections < 2 {
		log.Fatal("Minimum value for maxConnections is 2")
		return
	}

	requestInterval = *flagRequestInterval

	var err error
	robot, err = robots.New(protocol+"://"+domain+"/robots.txt", userAgent)
	if err == nil {
		robot.Read()
	} else {
		log.Println("Robots.txt error:", err)
	}

	explorer = explorers.New(userAgent, explorers.Options{
		Domain:   domain,
		Protocol: protocol,
		Ethical:  *flagIsUnethical,
	})

	// Init lock
	lock = semaphore.NewWeighted(int64(maxConnections))

	go waitForURLToScan()
	go info()

	// Scan the url provided
	ch <- url

	time.Sleep(1 * time.Second)
	wg.Wait()

	time.Sleep(1 * time.Second)
	showScanStatus()

	generateSitemap()

	fmt.Println(explorer.GetBadPagesFound())
}
