package main

import (
	"os"

	"github.com/mattiasthalen/qlik-script-extractor/cmd"
)

func main() {
	root := cmd.NewRootCmd()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
