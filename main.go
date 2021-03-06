package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"quatermain/robots"
	"quatermain/url"
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

const httpGenericError = 1
const httpBodyParseError = 2
const httpXRobotTag = 3
const userAgent = "quatermain"

//go:embed help.txt
var helpTemplate string

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

// The request interval is a parameter used to avoid side effects into the website.
// For example is a website use geolocation free service with a limited call of 120 x minutes
// is better set the param as 1
var requestInterval = 0

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

// The waitgroup used to block the main loop
var wg = new(sync.WaitGroup)

// Robot instance to respect and check url based of robots.txt
var robot robots.IRobot

/*
getPage fetch the html page and return an error
in case of the following conditions

- Ok                                    -> CODE 0
- Request generic error                 -> CODE 1
- Cannot read the body                  -> CODE 2
- Response headers contains x-robot-tag -> code 3
- Response status code ! 200            -> CODE xxx
*/
func getPage(url string) (*goquery.Document, int, error) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, httpGenericError, err
	}

	request.Header.Set("User-Agent", userAgent)

	response, err := client.Do(request)
	if err != nil {
		return nil, httpGenericError, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, response.StatusCode, errors.New("Not a valid page")
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, httpBodyParseError, err
	}

	xRobotsTag := response.Header.Get("X-Robots-Tag")
	if strings.Contains(xRobotsTag, "noindex") == true || strings.Contains(xRobotsTag, "nofollow") == true {
		return nil, httpXRobotTag, errors.New("Page cannot be followed or indexed")
	}

	return doc, 0, nil
}

func scanPage(page *goquery.Document) {
	robotsNodes := page.Find("meta[name=\"robots\"]")
	robotsMetaContent, ok := robotsNodes.Attr("content")
	if ok == true {
		if strings.Contains(robotsMetaContent, "nofollow") == true || strings.Contains(robotsMetaContent, "noindex") == true {
			return
		}
	}

	page.Find("a").Each(func(i int, s *goquery.Selection) {

		link := url.New(s, url.Options{
			DecorateRelativeURLWithDomain:   domain,
			DecorateRelativeURLWithProtocol: protocol,
		})

		if link.EmptyHref() ||
			link.IsDowload() ||
			link.IsInDomain(domain) == false ||
			link.IsMedia() ||
			link.IsHash() ||
			link.HaveNoFollow() {
			return
		}

		linkWithoutHash := link.StripHash()

		if isURLJustFound(&allLinks, linkWithoutHash) == true {
			return
		}

		allLinks = append(allLinks, linkWithoutHash)

		go func() {
			ch <- linkWithoutHash
		}()

	})
}

func start(url string) {

	if robot != nil && robot.CheckURL(url) == false {
		return
	}

	if isURLJustFound(&linksSuccessed, url) == true {
		return
	}

	page, statusCode, err := getPage(url)

	if err != nil {
		// Max connections created.
		// try to repeat
		if strings.Contains(err.Error(), "too many open files") {
			go func() {
				time.Sleep(time.Second)
				ch <- url
			}()
			return
		}

		appendBadURL(url, statusCode)
		return
	}

	linksSuccessed = append(linksSuccessed, url)
	go scanPage(page)
}

func checkLifeActivity() {
	for {
		showScanStatus()
		time.Sleep(time.Second)
	}
}

func waitForURLToScan() {
	for urlToScan := range ch {
		lock.Acquire(ctx, 1)
		wg.Add(1)

		time.Sleep(time.Duration(requestInterval) * time.Second)

		// Eexecute the page fetching inside an anon function
		// In this case we can take advantage of defer logics inside a main loop
		go func(url string) {
			defer decreaseConnectionsOpened()
			defer lock.Release(1)
			defer func() {
				time.Sleep(1 * time.Second)
				wg.Done()
			}()

			increaseConnectionsOpened()
			start(url)
		}(urlToScan)
	}
}

// 1. Start scanning page from endpoint provided
// 2. Find all links (exclude duplicated ones)
// 3. For each link found go to page and start scanning again
// 4. The script finish when there are no more activities of scan
func main() {
	// Defer the closing of the channel
	defer close(ch)

	args := os.Args
	if len(args) <= 1 {
		panic("Missing URL argument")
	}

	url := args[len(args)-1]

	flagMaxConnections := flag.Int("c", maxConnections, "The allowed max connections")
	flagRequestInterval := flag.Int("i", requestInterval, "The interval to wait before a request")
	flagIsHelp := flag.Bool("h", false, "The help")
	flag.Parse()

	if *flagIsHelp == true {
		fmt.Println(helpTemplate)
		return
	}

	maxConnections = *flagMaxConnections
	requestInterval = *flagRequestInterval

	protocol = extractProtocol(url)
	domain = extractDomain(url)

	var err error
	robot, err = robots.New(protocol+"://"+domain+"/robots.txt", userAgent)
	if err == nil {
		robot.Read()
	} else {
		fmt.Println("Robots.txt error:", err)
	}

	// Init lock
	lock = semaphore.NewWeighted(int64(maxConnections))

	// wg.Add(1)

	go waitForURLToScan()

	// Scan the url provided
	ch <- url

	time.Sleep(2 * time.Second)
	wg.Wait()

	showScanStatus()

	if len(linksSuccessed) > 0 {
		generateSitemap()
	}

	for _, val := range linksFailed {
		fmt.Println(val.StatusCode, val.URL)
	}

}
