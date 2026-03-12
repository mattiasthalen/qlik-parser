package ui

import (
	"os"

	"golang.org/x/term"
)

// IsTTY reports whether f is a terminal (TTY).
func IsTTY(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
