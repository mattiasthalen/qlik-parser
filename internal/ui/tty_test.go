package ui_test

import (
	"os"
	"testing"

	"github.com/mattiasthalen/qlik-parser/internal/ui"
)

func TestIsTTY_NonTTY(t *testing.T) {
	f, err := os.CreateTemp("", "tty-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	defer func() { _ = f.Close() }()

	if ui.IsTTY(f) {
		t.Error("expected regular file to not be a TTY")
	}
}
