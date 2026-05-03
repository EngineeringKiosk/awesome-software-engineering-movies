package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "awesome-software-engineering-movies",
	Short: "Helper tooling for a curated list of software engineering movies",
	Long: `A curated list of movies, documentaries and other related material to watch
related to Software Engineering, Open Source, Programming languages and culture.
To help with this, we automate as much as possible.
This is the helper tooling around the movie collection.

More information at https://github.com/EngineeringKiosk/awesome-software-engineering-movies`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
