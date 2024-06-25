package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "musicgreed",
	Short: "A command-line tool to aid in collecting music.",
	Long: `MusicGreed aims to speed up efforts to build a complete digital music
collection. This is done by using "setcover" to calculate a collection goal for a
music artist, or "remainder" to calculate the set cover on the tracks missing from a
current collection (feature to come).`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

}
