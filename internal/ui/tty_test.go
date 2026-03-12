package ui_test

import (
	"os"
	"testing"

	"github.com/mattiasthalen/qlik-script-extractor/internal/ui"
)

func TestIsTTY_NonTTY(t *testing.T) {
	f, err := os.CreateTemp("", "tty-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	if ui.IsTTY(f) {
		t.Error("expected regular file to not be a TTY")
	}
}
