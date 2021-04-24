package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"time"
)

func info() {
	for {
		showScanStatus()
		time.Sleep(time.Second)
	}
}

func decreaseConnectionsOpened() {
	connectionsOpened = connectionsOpened - 1
}

func increaseConnectionsOpened() {
	connectionsOpened = connectionsOpened + 1
}

func extractDomain(url string) string {
	lastByteOfURL := url[len(url)-1]
	if string(lastByteOfURL) != "/" {
		url += "/"
	}

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

func showScanStatus() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	//c.Run()
	fmt.Println("Domain:", domain)
	fmt.Println("Protocol:", protocol)
	fmt.Println("Request interval:", requestInterval)
	fmt.Println(fmt.Sprintf("Connections: %d / %d", connectionsOpened, maxConnections))
	fmt.Println("URL Found Total:", len(explorer.GetPagesFound()))
	fmt.Println("URL Found NOK:", len(explorer.GetBadPagesFound()))
	fmt.Println("URL Found OK:", len(explorer.GetGoodPagesFound()))
}
