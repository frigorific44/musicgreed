package main

import (
	"log"

	"github.com/frigorific44/musicgreed/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	if err := doc.GenMarkdownTree(cmd.NewRootCmd(), "./"); err != nil {
		log.Fatal(err)
	}
}
