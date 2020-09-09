package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/miclle/gsd"
)

const (
	defaultPath = "./"             // default document source code path
	defaultAddr = "localhost:3000" // default webserver address
)

var (
	path = flag.String("path", defaultPath, "Document source code path")

	// network
	httpAddr = flag.String("http", defaultAddr, "HTTP service address")

	// build
	build = flag.Bool("build", false, "Build documents")
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: gsd -http="+defaultAddr+"\n")
	flag.PrintDefaults()
	os.Exit(2)
}

// Main docs
func main() {

	flag.Usage = usage
	flag.Parse()

	corpus := gsd.NewCorpus(*path)

	log.Printf("build: %#v", *build)

	// build
	if *build {
		if err := corpus.Export(); err != nil {
			log.Fatal(err)
		}
		return
	}

	// start document webserver
	err := corpus.Watch()
	if err != nil {
		log.Fatalf("Watch source code failed %v", err)
	}

	if err := http.ListenAndServe(*httpAddr, corpus); err != nil {
		log.Fatalf("ListenAndServe %s: %v", *httpAddr, err)
	}
}
