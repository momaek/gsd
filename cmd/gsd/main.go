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
	httpAddr = flag.String("http", defaultAddr, "HTTP service address")

	// watch mode
	watch = flag.Bool("watch", false, "Watch mode")
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

	// start document webserver
	if *watch {
		err := corpus.Watch(*httpAddr)
		if err != nil {
			log.Fatalf("Watch source code failed %v", err)
		}
	} else {
		// build
		if err := corpus.Export(); err != nil {
			log.Fatal(err)
		}
	}
}
