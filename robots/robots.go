package robots

import (
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

// IRobot ...
type IRobot interface {
	CheckURL(string) bool
	Read()
	GetRules() []Rule
	GetURL() string
}

// Rule ...
type Rule struct {
	isAllowed bool
	url       string
}

// Robot ...
type robot struct {
	url               string
	content           []byte
	rules             []Rule
	userAgent         string
	allowedUserAgents []string
}

func (r *robot) GetURL() string {
	return r.url
}

func (r *robot) GetRules() []Rule {
	return r.rules
}

// CheckURL control an URL provided
func (r *robot) CheckURL(url string) bool {
	isAllowed := true
	for _, rule := range r.rules {

		match, err := regexp.MatchString(rule.url, url)
		if err == nil && match == true {
			isAllowed = rule.isAllowed
		}
	}

	return isAllowed
}

func (r *robot) isAllowedUserAgent(userAgent string) bool {
	for _, ua := range r.allowedUserAgents {
		if ua == userAgent {
			return true
		}
	}
	return false
}

func (r *robot) isUserAgentRule(rule string) bool {
	return strings.Contains(rule, "User-agent:")
}

func (r *robot) extractUserAgentFromRule(rule string) string {
	return strings.Trim(strings.Replace(rule, "User-agent:", "", -1), " ")
}

func (r *robot) extractAllowedOrDisallowed(rule string) (bool, error) {
	if strings.Contains(rule, "Allow:") {
		return true, nil
	}

	if strings.Contains(rule, "Disallow:") {
		return false, nil
	}

	return true, errors.New("The rule is invalid")
}

func (r *robot) extractRuleFromLine(line string) (Rule, error) {
	var rule Rule

	isAllowed, err := r.extractAllowedOrDisallowed(line)
	if err != nil {
		return rule, err
	}

	urlRgx := regexp.MustCompile("^.*:")
	url := strings.Trim(urlRgx.ReplaceAllString(line, ""), " ")
	if url == "" {
		return rule, errors.New("Url is empty")
	}

	rule.isAllowed = isAllowed
	rule.url = url

	return rule, nil
}

// Read the content of the file and collect
// the rules for the user agent provided
func (r *robot) Read() {
	const globalUserAgent = "*"
	var globalRules []Rule
	var otherRules []Rule

	content := string(r.content)
	lines := strings.Split(content, "\n")

	userAgent := globalUserAgent

	for _, line := range lines {
		if r.isUserAgentRule(line) == true {
			userAgent = r.extractUserAgentFromRule(line)
			if r.isAllowedUserAgent(userAgent) == true {
				continue
			}
		}

		rule, err := r.extractRuleFromLine(line)
		if err != nil {
			continue
		}

		if userAgent == globalUserAgent {
			globalRules = append(globalRules, rule)
		} else {
			otherRules = append(otherRules, rule)
		}
	}

	r.rules = append(otherRules, globalRules...)

	return
}

func (r *robot) fetch() error {
	response, err := http.Get(r.url)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.New("Cannot fetch robots.txt")
	}

	rBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	r.content = rBytes

	return nil
}

// New .. iRobot.
func New(url string, userAgent string) (IRobot, error) {
	r := robot{
		url:       url,
		userAgent: userAgent,
	}

	r.allowedUserAgents = []string{"*"}
	if userAgent != "" {
		r.allowedUserAgents = append(r.allowedUserAgents, userAgent)
	}

	err := r.fetch()
	if err != nil {
		return nil, err
	}

	return &r, nil
}
