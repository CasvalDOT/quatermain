package url

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var pageExtensions = []string{
	"html",
	"asp",
	"php",
	"",
}

// Options ...
type Options struct {
	DecorateRelativeURLWithProtocol string
	DecorateRelativeURLWithDomain   string
}

type urlDOM struct {
	element   *goquery.Selection
	href      string
	extension string
	rel       string
	target    string
	protocol  string
	domain    string
}

// IURLDOM interface
type IURLDOM interface {
	EmptyHref() bool
	IsRelativeURL() bool
	HaveNoFollow() bool
	IsHash() bool
	IsInDomain(string) bool
	IsMedia() bool
	IsDowload() bool
	StripHash() string
}

func (u *urlDOM) matchFirstChar(charToMatch string) bool {
	if u.href == "" {
		return false
	}
	firstChar := string(u.href[0])
	return charToMatch == firstChar
}

func (u *urlDOM) IsRelativeURL() bool {
	return u.matchFirstChar("/")
}

func (u *urlDOM) IsHash() bool {
	return u.matchFirstChar("#")
}

func (u *urlDOM) EmptyHref() bool {
	return u.href == ""
}

func (u *urlDOM) HaveNoFollow() bool {
	return strings.Contains(u.rel, "nofollow")
}

func (u *urlDOM) IsMedia() bool {
	for _, extension := range pageExtensions {
		if extension == u.extension {
			return false
		}
	}

	return true
}

func (u *urlDOM) IsDowload() bool {
	_, exist := u.element.Attr("download")
	return exist
}

func (u *urlDOM) IsInDomain(domain string) bool {
	allowedPattern := []string{
		"https://www." + domain,
		"https://" + domain,
		"http://www." + domain,
		"http://" + domain,
	}
	rgx := regexp.MustCompile("^(" + strings.Join(allowedPattern, "|") + ")")
	return rgx.MatchString(u.href)
}

func (u *urlDOM) StripHash() string {
	rgx := regexp.MustCompile("#.*")
	return rgx.ReplaceAllString(u.href, "")
}

func (u *urlDOM) extractExtension() {
	rgx := regexp.MustCompile("^.*(\\.|)$")
	u.extension = rgx.ReplaceAllString(u.href, "")
}

func (u *urlDOM) extractAttribute(attribute string) string {
	attributeValue, exist := u.element.Attr(attribute)
	if exist == false {
		return ""
	}

	return attributeValue
}

func (u *urlDOM) extractRelAttribute() {
	u.rel = u.extractAttribute("rel")
}

func (u *urlDOM) extractHref() {
	u.href = u.extractAttribute("href")
}

func (u *urlDOM) init() {
	u.extractRelAttribute()
	u.extractExtension()
	u.extractHref()

}

// New ...
func New(url *goquery.Selection, options Options) IURLDOM {
	instance := urlDOM{
		element: url,
	}

	instance.init()

	if instance.IsRelativeURL() {
		instance.protocol = options.DecorateRelativeURLWithProtocol
		instance.domain = options.DecorateRelativeURLWithDomain
		instance.href = instance.protocol + "://" + instance.domain + instance.href
	}

	return &instance
}
