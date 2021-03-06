package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
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

// Links that did not return a status code 200 or that generated some error end up in this variable
var linksFailed = []BadURL{}

// Instead if a URL is valid, will store here
var linksSuccessed = []string{}

// In any case all links are contained here
var allLinks = []string{}

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

/*
	getPage fetch the html page

	About errors:
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
		return nil, response.StatusCode, errors.New("Response status code is not valid")
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, httpBodyParseError, err
	}

	xRobotsTag := response.Header.Get("X-Robots-Tag")
	if haveNoIndexOrNoFollow(xRobotsTag) == true {
		return nil, httpXRobotTag, errors.New("Page cannot be followed or indexed")
	}

	return doc, 0, nil
}

func scanPage(page *goquery.Document) {
	robotsNodes := page.Find("meta[name=\"robots\"]")
	robotsMetaContent, ok := robotsNodes.Attr("content")
	if ok == true && haveNoIndexOrNoFollow(robotsMetaContent) {
		return
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

	defer func() {
		time.Sleep(1 * time.Second)
		wg.Done()
	}()

	if robot != nil && robot.CheckURL(url) == false {
		return
	}

	if isURLJustFound(&linksSuccessed, url) == true {
		return
	}

	page, statusCode, err := getPage(url)

	if err != nil {
		// Max connections created. try to repeat the request
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

func info() {
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

		go func(url string) {
			defer decreaseConnectionsOpened()
			defer lock.Release(1)

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
	flagMaxConnections := flag.Int("c", maxConnections, "The allowed max connections.")
	flagRequestInterval := flag.Float64("i", requestInterval, "The interval to wait before a request")
	flagIsHelp := flag.Bool("h", false, "Print the help")
	flag.Parse()

	if *flagIsHelp == true {
		fmt.Println(helpTemplate)
		return
	}

	// Defer the closing of the channel
	defer close(ch)

	url := getURLFromArguments(os.Args)
	if url == "" {
		log.Fatalln("Missing URL argument")
		return
	}

	maxConnections = *flagMaxConnections
	requestInterval = *flagRequestInterval

	if maxConnections < 2 {
		log.Fatal("Minmum value for maxConnections is 2")
		return
	}

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

	go waitForURLToScan()
	go info()

	// Scan the url provided
	ch <- url

	time.Sleep(2 * time.Second)
	wg.Wait()

	time.Sleep(1 * time.Second)
	showScanStatus()

	if len(linksSuccessed) > 0 {
		generateSitemap()
	}

	for _, val := range linksFailed {
		fmt.Println(val.StatusCode, val.URL)
	}

}
