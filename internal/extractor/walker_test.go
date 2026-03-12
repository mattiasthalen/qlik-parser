package extractor_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/mattiasthalen/qlik-script-extractor/internal/extractor"
)

func TestWalkFindsQVWFiles(t *testing.T) {
	root := t.TempDir()
	dirs := []string{
		filepath.Join(root, "a"),
		filepath.Join(root, "b", "c"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	files := []string{
		filepath.Join(root, "top.qvw"),
		filepath.Join(root, "a", "first.qvw"),
		filepath.Join(root, "b", "c", "deep.qvw"),
		filepath.Join(root, "a", "ignore.txt"),
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte{0x00}, 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, warns := extractor.Walk(root)

	sort.Strings(got)
	expected := []string{
		filepath.Join(root, "a", "first.qvw"),
		filepath.Join(root, "b", "c", "deep.qvw"),
		filepath.Join(root, "top.qvw"),
	}
	sort.Strings(expected)

	if len(warns) != 0 {
		t.Errorf("expected no warns, got %v", warns)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d files, got %d: %v", len(expected), len(got), got)
	}
	for i := range got {
		if got[i] != expected[i] {
			t.Errorf("index %d: expected %s got %s", i, expected[i], got[i])
		}
	}
}

func TestWalkIgnoresNonQVW(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.qvf", "b.txt", "c.qvs", "d.QVW"} {
		_ = os.WriteFile(filepath.Join(root, name), []byte{0x00}, 0644)
	}
	got, _ := extractor.Walk(root)
	if len(got) != 0 {
		t.Errorf("expected 0 files, got: %v", got)
	}
}

func TestWalkUnreadableSubdirEmitsWarn(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root — permission tests unreliable")
	}
	root := t.TempDir()
	denied := filepath.Join(root, "denied")
	if err := os.MkdirAll(denied, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(denied, 0755) })

	_, warns := extractor.Walk(root)
	if len(warns) == 0 {
		t.Error("expected at least one warn for unreadable subdir, got none")
	}
}

func TestWalkDoesNotFollowSymlinks(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	_ = os.WriteFile(filepath.Join(target, "linked.qvw"), []byte{0x00}, 0644)
	_ = os.Symlink(target, filepath.Join(root, "link"))

	got, _ := extractor.Walk(root)
	for _, f := range got {
		if filepath.Base(f) == "linked.qvw" {
			t.Error("Walk followed a symlink — expected it to be skipped")
		}
	}
}
