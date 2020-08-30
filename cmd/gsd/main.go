package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/miclle/gsd"
	"github.com/yosssi/gohtml"
)

// Main docs
func main() {

	corpus := gsd.NewCorpus()

	err := corpus.Init()
	if err != nil {
		log.Fatal(err)
		return
	}

	page := gsd.NewPage(corpus)

	data := map[string]interface{}{
		"tree": corpus.Tree,
	}

	var buf bytes.Buffer
	if err := page.SidebarHTML.Execute(&buf, data); err != nil {
		log.Printf("%s.Execute: %s", page.SidebarHTML.Name(), err)
	}
	sidebar := gohtml.FormatBytes(buf.Bytes())

	err = os.MkdirAll("docs", os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile("docs/sidebar.html", sidebar, 0644)
	if err != nil {
		log.Fatal(err)
	}

	for _, pkg := range corpus.Packages {

		var (
			data, err = gsd.RenderPackage(page, pkg)
			path      = strings.TrimPrefix(pkg.ImportPath, pkg.Module.Path)
		)

		path = fmt.Sprintf("docs/%s", path)

		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}

		filename := fmt.Sprintf("%s/index.html", path)

		err = ioutil.WriteFile(filename, data, 0644)
		if err != nil {
			log.Fatal(err)
		}
	}

}
