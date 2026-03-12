//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/mattiasthalen/qlik-parser/internal/extractor"
)

func main() {
	script, err := extractor.ExtractScript("internal/extractor/testdata/fixtures/valid.qvw")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile("internal/extractor/testdata/fixtures/valid.qvs.golden", []byte(script), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing golden: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Generated valid.qvs.golden")
}
