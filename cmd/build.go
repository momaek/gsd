package cmd

import (
	"log"

	"github.com/miclle/gsd/document"
	"github.com/spf13/cobra"
)

const defaultOutputPath = "./docs" // default document export path

// Document source code path
var output string

// buildCmd represents the start command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Generate documents",
	Run: func(cmd *cobra.Command, args []string) {

		config := &document.Config{
			Path:   path,
			Output: output,
		}

		corpus, err := document.NewCorpus(config)
		if err != nil {
			log.Fatal(err)
		}

		if err := corpus.Export(); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	buildCmd.PersistentFlags().StringVarP(&output, "output", "o", defaultOutputPath, "Document source code path")

	rootCmd.AddCommand(buildCmd)
}
