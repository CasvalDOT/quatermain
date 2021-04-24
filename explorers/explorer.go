package explorers

import (
	"errors"
	"fmt"
	"net/http"
	netURL "net/url"
	"quatermain/url"
	"regexp"

	"github.com/PuerkitoBio/goquery"
)

// ErrorCode ...
const ErrorCode = 1

// NoFollowCode ...
const NoFollowCode = 2

// Page ...
type Page struct {
	StatusCode    int
	Link          string
	CanonicalLink string
}

// Options ...
type Options struct {
	Domain   string
	Protocol string
	Ethical  bool
}

type explorer struct {
	ethical    bool
	name       string
	domain     string
	protocol   string
	pagesFound []Page
	pageCache  []string
	channel    chan string
}

// IExplorer ...
type IExplorer interface {
	Fetch(string) (*goquery.Document, int, error)
	IsPageVisited(string) bool
	SearchLinks(*goquery.Document, func(string))
	GetPagesFound() []Page
	GetGoodPagesFound() []Page
	GetBadPagesFound() []Page
	BlockedByRobotsTag(*goquery.Document) bool
	FindCanonical(*goquery.Document) string
	AppendPage(*Page)
}

func (e *explorer) GetPagesFound() []Page {
	return e.pagesFound
}

func (e *explorer) GetGoodPagesFound() []Page {
	var response = []Page{}
	for _, page := range e.pagesFound {
		if page.StatusCode == 0 {
			response = append(response, page)
		}
	}

	return response
}

func (e *explorer) GetBadPagesFound() []Page {
	var response = []Page{}
	for _, page := range e.pagesFound {
		if page.StatusCode != 0 {
			response = append(response, page)
		}
	}

	return response
}

func (e *explorer) AppendPage(page *Page) {
	e.pagesFound = append(e.pagesFound, *page)
}

func (e *explorer) IsPageVisited(pageURL string) bool {
	for _, page := range e.pagesFound {
		if pageURL == page.Link {
			return true
		}
	}

	return false
}

func (e *explorer) isLinkInCache(link string) bool {
	for _, l := range e.pageCache {
		if link == l {
			return true
		}
	}

	return false
}

func (e *explorer) Fetch(linkToPage string) (*goquery.Document, int, error) {
	client := &http.Client{}

	baseURL, err := netURL.Parse(linkToPage)
	if err != nil {
		fmt.Println("Malformed URL: ", err.Error(), baseURL)
	}
	r := regexp.MustCompile(" ")
	linkToPage = r.ReplaceAllString(linkToPage, "%20")

	request, err := http.NewRequest("GET", linkToPage, nil)
	if err != nil {
		return nil, ErrorCode, err
	}

	request.Header.Set("User-Agent", e.name)

	response, err := client.Do(request)
	if err != nil {
		return nil, ErrorCode, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, response.StatusCode, errors.New("Response status code is not valid")
	}

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, ErrorCode, err
	}

	xRobotsTag := response.Header.Get("X-Robots-Tag")
	if e.ethical == true && haveNoIndexOrNoFollow(xRobotsTag) == true {
		return nil, NoFollowCode, errors.New("Page cannot be followed or indexed")
	}

	return document, 0, nil
}

func (e *explorer) SearchLinks(page *goquery.Document, callback func(string)) {
	page.Find("a").Each(func(i int, s *goquery.Selection) {
		link := url.New(s, url.Options{
			DecorateRelativeURLWithDomain:   e.domain,
			DecorateRelativeURLWithProtocol: e.protocol,
		})

		if link.EmptyHref() ||
			link.IsDowload() ||
			link.IsInDomain(e.domain) == false ||
			link.IsMedia() ||
			link.IsHash() ||
			link.IsPhoneNumber() ||
			link.IsEmail() {
			return
		}

		if e.ethical && link.HaveNoFollow() {
			return
		}

		linkCleaned := link.GetHref()
		if e.isLinkInCache(linkCleaned) == true {
			return
		}

		e.pageCache = append(e.pageCache, linkCleaned)

		callback(linkCleaned)
	})
}

func (e *explorer) BlockedByRobotsTag(page *goquery.Document) bool {
	robotsNodes := page.Find("meta[name=\"robots\"]")
	robotsMetaContent, ok := robotsNodes.Attr("content")
	if ok == true && haveNoIndexOrNoFollow(robotsMetaContent) {
		return true
	}

	return false
}

func (e *explorer) FindCanonical(page *goquery.Document) string {
	canonicalNodes := page.Find("link[rel=\"canonical\"]")
	value, ok := canonicalNodes.Attr("href")
	if ok == false {
		value = ""
	}

	return value
}

// New ...
func New(name string, options Options) IExplorer {
	return &explorer{
		ethical:  options.Ethical,
		domain:   options.Domain,
		protocol: options.Protocol,
		name:     name,
	}
}
