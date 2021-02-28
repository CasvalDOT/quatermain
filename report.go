package main

import (
	"bytes"
	"os"
	"text/template"
	"time"
)

func generateSitemap() {

	templateData := map[string]interface{}{
		"List": linksSuccessed,
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
