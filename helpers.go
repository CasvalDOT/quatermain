package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func isEmptyURL(url string) bool {
	if url == "" {
		return true
	}
	return false
}

func decorateWithDomain(url string) string {
	return protocol + "://" + domain + url
}

func isRelativeURL(url string) bool {
	firstChar := string(url[0])
	matchChar := "/"
	if firstChar == matchChar {
		return true
	}

	return false
}

func isHashURL(url string) bool {
	firstChar := string(url[0])
	matchChar := "#"
	if firstChar == matchChar {
		return true
	}

	return false
}

func isURLJustFound(list *[]string, url string) bool {
	match := false
	for _, u := range *list {
		if u == url {
			match = true
			break
		}
	}
	return match
}

func isSameDomain(url string) bool {
	allowedPattern := []string{
		protocol + "://www." + domain,
		protocol + "://" + domain,
	}
	rgx := regexp.MustCompile("^(" + strings.Join(allowedPattern, "|") + ")")
	return rgx.MatchString(url)
}

func isMedia(url string) bool {
	rgx := regexp.MustCompile("(?i)\\.(pdf|docx|jpg|jpeg|webp|png|gif|mp4|avi|mkv|txt|xml|json)$")
	return rgx.MatchString(url)
}

func stripHash(url string) string {
	rgx := regexp.MustCompile("#.*")
	return rgx.ReplaceAllString(url, "")
}

func isAllowedURL(url string) bool {

	if isEmptyURL(url) == true {
		return false
	}

	if isHashURL(url) == true {
		return false
	}

	if isMedia(url) == true {
		return false
	}

	if isSameDomain(url) == false && isRelativeURL(url) == false {
		return false
	}

	return true

}

func decorateURL(url string) string {
	if isRelativeURL(url) {
		url = decorateWithDomain(url)
	}

	url = stripHash(url)

	return url
}

func extractDomain(url string) string {
	rgx := regexp.MustCompile("^" + protocol + "://(www\\.|)(.*?)\\/")
	matchs := rgx.FindAllStringSubmatch(url, 1)

	firstLayer := matchs[0]

	return firstLayer[len(firstLayer)-1]
}

func extractProtocol(url string) string {
	rgx := regexp.MustCompile("^(.*?):")
	matchs := rgx.FindAllStringSubmatch(url, 1)

	firstLayer := matchs[0]

	return firstLayer[len(firstLayer)-1]
}

func showScanStatus() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
	fmt.Println("Heart beat:", hearthBeatInterval)
	fmt.Println(fmt.Sprintf("Connections: %d / %d", connectionsOpened, maxConnections))
	fmt.Println("URL Found Total:", len(allLinks))
	fmt.Println("URL Found NOK:", len(linksFailed))
	fmt.Println("URL Found OK:", len(linksSuccessed))
}
