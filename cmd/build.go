package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// buildCmd represents the start command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Generate documentations",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Generate documentation")
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
