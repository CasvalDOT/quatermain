package explorers

import "strings"

func haveNoIndexOrNoFollow(content string) bool {
	return strings.Contains(content, "noindex") == true || strings.Contains(content, "nofollow") == true
}
