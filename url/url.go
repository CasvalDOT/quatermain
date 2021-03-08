package url

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var pageExtensions = []string{
	".html",
	".asp",
	".php",
	".page",
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
	GetHref() string
	EmptyHref() bool
	IsRelativeURL() bool
	HaveNoFollow() bool
	IsHash() bool
	IsInDomain(string) bool
	IsMedia() bool
	IsPhoneNumber() bool
	IsEmail() bool
	IsDowload() bool
}

func (u *urlDOM) matchFirstChar(charToMatch string) bool {
	if u.href == "" {
		return false
	}
	firstChar := string(u.href[0])
	return charToMatch == firstChar
}

func (u *urlDOM) GetHref() string {
	return u.href
}

func (u *urlDOM) isWithoutProtocol() bool {
	match, ok := regexp.MatchString("^//", u.href)
	return ok == nil && match == true
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

func (u *urlDOM) removeHash() string {
	rgx := regexp.MustCompile("#.*")
	return rgx.ReplaceAllString(u.href, "")
}

func (u *urlDOM) removeSpecialChars() string {
	rgx := regexp.MustCompile("\\n|\\r")
	return rgx.ReplaceAllString(u.href, "")
}

func (u *urlDOM) cleanHref() {
	s := u.removeHash()
	s = u.removeSpecialChars()
	u.href = s
}

func (u *urlDOM) IsPhoneNumber() bool {
	return strings.Contains(u.href, "tel:")
}

func (u *urlDOM) IsEmail() bool {
	return strings.Contains(u.href, "mailto:")
}

func (u *urlDOM) extractExtension() {
	rgx := regexp.MustCompile(u.domain + ".*(\\.[a-zA-Z0-9]+$)")
	matchs := rgx.FindAllStringSubmatch(u.href, -1)
	if len(matchs) == 0 {
		u.extension = ""
	} else {
		u.extension = matchs[0][1]
	}
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

func (u *urlDOM) init() {
	u.extractRelAttribute()
	u.extractExtension()
}

// New ...
func New(url *goquery.Selection, options Options) IURLDOM {
	instance := urlDOM{
		element: url,
	}

	instance.protocol = options.DecorateRelativeURLWithProtocol
	instance.domain = options.DecorateRelativeURLWithDomain
	instance.href = instance.extractAttribute("href")

	if instance.isWithoutProtocol() {
		instance.href = instance.protocol + ":" + instance.href
	}

	if instance.IsRelativeURL() {
		instance.href = instance.protocol + "://" + instance.domain + instance.extractAttribute("href")
	}

	instance.init()
	instance.cleanHref()

	return &instance
}
