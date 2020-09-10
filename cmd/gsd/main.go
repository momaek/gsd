package main

import (
	"flag"
	"fmt"
	"log"
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
	httpAddr = flag.String("http", "", "HTTP service address (e.g., '127.0.0.1:3000' or just ':3000')")
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

	if *httpAddr != "" {
		// start document webserver
		if err := corpus.Watch(*httpAddr); err != nil {
			log.Fatalf("Watch source code failed %v", err)
		}
		return
	}

	// build
	if err := corpus.Export(); err != nil {
		log.Fatal(err)
	}
}
