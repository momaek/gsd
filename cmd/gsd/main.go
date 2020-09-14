package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/miclle/gsd"
)

const (
	defaultPath       = "./"             // default document source code path
	defaultOutputPath = "./docs"         // default document export path
	defaultAddr       = "localhost:3000" // default webserver address
)

var (
	path = flag.String("path", defaultPath, "Document source code path")

	output = flag.String("output", defaultOutputPath, "Document source code path")

	// network
	httpAddr = flag.String("http", "", "HTTP service address (e.g., '127.0.0.1:3000' or just ':3000')")
)

func usage() {
	fmt.Fprintf(os.Stderr, "version: "+gsd.Version+"\n")
	fmt.Fprintf(os.Stderr, "usage:\n")
	fmt.Fprintf(os.Stderr, "  generate documentations:\n\tgsd\n")
	fmt.Fprintf(os.Stderr, "  start documentation webserver:\n\tgsd -http="+defaultAddr+"\n")
	fmt.Fprintf(os.Stderr, "options:\n")
	flag.PrintDefaults()
	os.Exit(0)
}

// Main docs
func main() {

	flag.Usage = usage
	flag.Parse()

	config := &gsd.Config{
		Path: *path,
		Addr: *httpAddr,
	}

	corpus, err := gsd.NewCorpus(config)
	if err != nil {
		log.Fatal(err)
	}

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
