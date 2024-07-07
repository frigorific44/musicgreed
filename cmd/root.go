package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "musicgreed",
		Short: "A command-line tool to aid in collecting music.",
		Long: `MusicGreed aims to speed up efforts to build a complete digital music
collection. This is done by using "setcover" to calculate a collection goal for a
music artist, or "remainder" to calculate the set cover on the tracks missing from a
current collection (feature to come).`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Configure the logger
			if lout, _ := cmd.Flags().GetString("output"); lout != "" {
				file, err := os.OpenFile(lout, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
				if err == nil {
					slog.SetDefault(slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{Level: slog.LevelDebug})))
					return
				}
				slog.Error(
					"opening output file returned an error",
					"output_path", lout,
				)
			}
			slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
		},
	}

	cmd.PersistentFlags().StringP("output", "o", "", "path to log output file")

	cmd.AddCommand(
		NewSetCoverCmd(),
	)

	cmd.CompletionOptions.HiddenDefaultCmd = true

	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(rootCmd *cobra.Command) {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
