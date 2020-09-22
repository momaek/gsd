package cmd

import (
	"log"

	"github.com/miclle/gsd"
	"github.com/spf13/cobra"
)

const defaultOutputPath = "./docs" // default document export path

// Document source code path
var output string

// buildCmd represents the start command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Generate documentations",
	Run: func(cmd *cobra.Command, args []string) {
		config := &gsd.Config{
			Path:   path,
			Output: output,
		}

		corpus, err := gsd.NewCorpus(config)
		if err != nil {
			log.Fatal(err)
		}

		if err := corpus.Export(); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	buildCmd.PersistentFlags().StringVar(&output, "output", defaultOutputPath, "Document source code path")

	rootCmd.AddCommand(buildCmd)
}
