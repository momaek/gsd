package cmd

import (
	"fmt"
	"log"

	"github.com/miclle/gsd/document"
	"github.com/spf13/cobra"
)

const (
	defaultAddr = "localhost:3000" // default webserver address

	defaultAutoOpenBrowser = true // default auto open browser when webserver startup
)

// http server address
var httpAddr string

// Auto open browser when webserver startup
var autoOpenBrowser bool

// serveCmd represents the start command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start documentation webserver",
	Long:  "Start documentation webserver:\n\tgsd -http=" + defaultAddr + "\n",
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println(path, excludes)

		config := &document.Config{
			Path:            path,
			Addr:            httpAddr,
			AutoOpenBrowser: autoOpenBrowser,
		}

		corpus, err := document.NewCorpus(config)
		if err != nil {
			log.Fatal(err)
		}

		if err := corpus.Watch(httpAddr); err != nil {
			log.Fatal(err)
		}

	},
}

func init() {
	serveCmd.PersistentFlags().StringVar(&httpAddr, "http", defaultAddr, "HTTP service address (e.g., '127.0.0.1:3000' or just ':3000')")
	serveCmd.PersistentFlags().BoolVar(&autoOpenBrowser, "open", defaultAutoOpenBrowser, "Auto open browser when webserver startup")

	rootCmd.AddCommand(serveCmd)
}
