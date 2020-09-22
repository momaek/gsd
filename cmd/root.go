package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	defaultPath = "./" // default document source code path
)

// Document source code path
var path string

// exclude paths
var excludes []string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gsd",
	Short: "Generate documentations with source code comment",
	Long: `Generate documentations with source code comment

	`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&path, "path", defaultPath, "Document source code path")
	rootCmd.PersistentFlags().StringSliceVar(&excludes, "e", []string{}, "Exclude paths")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

}
