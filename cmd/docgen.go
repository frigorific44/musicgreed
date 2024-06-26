package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// docgenCmd represents the docgen command
var docgenCmd = &cobra.Command{
	Use:    "docgen",
	Short:  "generate the Markdown documentation for MusicGreed",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("docgen called")
		if err := doc.GenMarkdownTree(rootCmd, "./docs"); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(docgenCmd)
}
