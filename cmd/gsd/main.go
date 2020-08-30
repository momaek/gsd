package main

import (
	"log"

	"github.com/miclle/gsd"
)

// Main docs
func main() {

	corpus := gsd.NewCorpus()

	err := corpus.Init()
	if err != nil {
		log.Fatal(err)
		return
	}

	err = corpus.RenderStaticAssets()
	if err != nil {
		log.Fatal(err)
		return
	}

	for _, pkg := range corpus.Packages {
		corpus.RenderPackage(pkg)
	}
}
