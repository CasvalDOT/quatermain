package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func decreaseConnectionsOpened() {
	connectionsOpened = connectionsOpened - 1
}

func increaseConnectionsOpened() {
	connectionsOpened = connectionsOpened + 1
}

func appendBadURL(badURL string, errorCode int) {
	linksFailed = append(linksFailed, Page{
		Link:          badURL,
		StatusCode:    errorCode,
		CanonicalLink: "",
	})
}

func isURLJustFound(list *[]Page, url string) bool {
	match := false
	for _, u := range *list {
		if u.Link == url {
			match = true
			break
		}
	}
	return match
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

func getURLFromArguments(args []string) string {
	onlyArgs := args[1:]
	for _, arg := range onlyArgs {
		match, err := regexp.MatchString("http(s|)://", arg)
		if err != nil || match == false {
			continue
		}

		return arg
	}

	return ""
}

func haveNoIndexOrNoFollow(content string) bool {
	return strings.Contains(content, "noindex") == true || strings.Contains(content, "nofollow") == true
}

func showScanStatus() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
	fmt.Println("Domain:", domain)
	fmt.Println("Protocol:", protocol)
	fmt.Println("Request interval:", requestInterval)
	fmt.Println(fmt.Sprintf("Connections: %d / %d", connectionsOpened, maxConnections))
	fmt.Println("URL Found Total:", len(allLinks))
	fmt.Println("URL Found NOK:", len(linksFailed))
	fmt.Println("URL Found OK:", len(linksSuccessed))
}
