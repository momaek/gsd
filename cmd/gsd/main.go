package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/miclle/gsd"
)

// Main docs
func main() {

	corpus := gsd.NewCorpus()

	err := corpus.Init()
	if err != nil {
		os.Exit(2)
		return
	}

	presentation := gsd.NewPresentation(corpus)

	data := map[string]interface{}{
		"tree": corpus.Tree,
	}

	var buf bytes.Buffer
	if err := presentation.SidebarHTML.Execute(&buf, data); err != nil {
		log.Printf("%s.Execute: %s", presentation.SidebarHTML.Name(), err)
	}

	sidebar := buf.Bytes()

	err = os.MkdirAll("docs", os.ModePerm)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("docs/sidebar.html", sidebar, 0644)
	if err != nil {
		panic(err)
	}

	for name, pkg := range corpus.Packages {
		fmt.Printf("name: %s,\t%p\n", name, pkg)

		data, err := gsd.RenderPackage(presentation, pkg)

		// filename := strings.ReplaceAll(name, "/", "-")

		path := strings.TrimPrefix(pkg.ImportPath, pkg.Module.Path)

		path = fmt.Sprintf("docs/%s", path)
		os.MkdirAll(path, os.ModePerm)

		filename := fmt.Sprintf("%s/index.html", path)

		fmt.Println("filename:", filename)

		err = ioutil.WriteFile(filename, data, 0644)
		if err != nil {
			panic(err)
		}
	}

}
