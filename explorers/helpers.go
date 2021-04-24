package explorers

import "strings"

func haveNoIndex(content string) bool {
	return strings.Contains(content, "noindex")
}

func haveNoFollow(content string) bool {
	return strings.Contains(content, "nofollow")
}

func haveNoIndexOrNoFollow(content string) bool {
	return haveNoIndex(content) || haveNoFollow(content)
}
