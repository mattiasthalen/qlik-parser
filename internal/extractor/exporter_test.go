package extractor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mattiasthalen/qlik-script-extractor/internal/extractor"
)

func TestResolveOutputPath_Alongside(t *testing.T) {
	qvwPath := "/data/etl/sales.qvw"
	got := extractor.ResolveOutputPath(qvwPath, "/data", "")
	want := "/data/etl/sales.qvs"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestResolveOutputPath_Mirror(t *testing.T) {
	qvwPath := "/data/etl/sales.qvw"
	got := extractor.ResolveOutputPath(qvwPath, "/data", "/out")
	want := "/out/etl/sales.qvs"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestResolveOutputPath_OutEqualSource(t *testing.T) {
	qvwPath := "/data/etl/sales.qvw"
	got := extractor.ResolveOutputPath(qvwPath, "/data", "/data")
	want := "/data/etl/sales.qvs"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestWriteScript_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "sub", "output.qvs")
	err := extractor.WriteScript(outPath, "/// LOAD * FROM t;", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if string(content) != "/// LOAD * FROM t;" {
		t.Errorf("unexpected content: %q", string(content))
	}
}

func TestWriteScript_DryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.qvs")
	err := extractor.WriteScript(outPath, "/// LOAD * FROM t;", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Error("expected file NOT to exist in dry-run mode")
	}
}

func TestWriteScript_CreatesIntermediateDirs(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "a", "b", "c", "output.qvs")
	if err := extractor.WriteScript(deep, "///", false); err != nil {
		t.Fatalf("expected dirs to be auto-created, got: %v", err)
	}
	if _, err := os.Stat(deep); err != nil {
		t.Errorf("file not found after WriteScript: %v", err)
	}
}
