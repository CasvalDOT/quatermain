package main

import (
	"bytes"
	"os"
	"text/template"
	"time"
)

func isElementUnique(list *[]string, element string) bool {
	match := true
	for _, u := range *list {
		if u == element {
			match = false
			break
		}
	}
	return match
}

func parseLinks() []string {
	var parsedPageLinks []string
	for _, page := range explorer.GetGoodPagesFound() {
		candidateLink := page.Link
		if page.CanonicalLink != "" {
			candidateLink = page.CanonicalLink
		}

		if isElementUnique(&parsedPageLinks, candidateLink) == false {
			continue
		}

		parsedPageLinks = append(parsedPageLinks, candidateLink)
	}

	return parsedPageLinks
}

func generateSitemap() {

	templateData := map[string]interface{}{
		"List": parseLinks(),
		"Time": time.Now().UTC().Format("2006-01-02T15:04:05-0700"),
	}

	tmpl := template.Must(template.New("sitemap").Parse(sitemapTemplate))

	var outputReader bytes.Buffer
	err := tmpl.Execute(&outputReader, templateData)

	if err != nil {
		panic(err)
	}

	file, err := os.Create("sitemap.xml")
	if err != nil {
		panic(err)
	}

	defer file.Close()

	file.Write(outputReader.Bytes())
}
