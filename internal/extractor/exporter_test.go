package extractor_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mattiasthalen/qlik-parser/internal/extractor"
)

func TestResolveOutputDir_QVW_Alongside(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix path test not applicable on Windows")
	}
	got := extractor.ResolveOutputDir("/data/etl/sales.qvw", "/data", "")
	want := "/data/etl/sales.qvw"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestResolveOutputDir_QVW_Mirror(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix path test not applicable on Windows")
	}
	got := extractor.ResolveOutputDir("/data/etl/sales.qvw", "/data", "/out")
	want := "/out/etl/sales.qvw"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestResolveOutputDir_OutEqualSource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix path test not applicable on Windows")
	}
	got := extractor.ResolveOutputDir("/data/etl/sales.qvw", "/data", "/data")
	want := "/data/etl/sales.qvw"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestResolveOutputDir_QVF_Alongside(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix path test not applicable on Windows")
	}
	got := extractor.ResolveOutputDir("/data/etl/app.qvf", "/data", "")
	want := "/data/etl/app.qvf"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestResolveOutputDir_QVF_Mirror(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix path test not applicable on Windows")
	}
	got := extractor.ResolveOutputDir("/data/etl/app.qvf", "/data", "/out")
	want := "/out/etl/app.qvf"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestWriteArtifacts_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "sub", "sales.qvw")
	artifacts := []extractor.Artifact{
		{Name: "script.qvs", Content: []byte("/// LOAD * FROM t;")},
		{Name: "measures.json", Content: []byte("[]")},
	}
	if err := extractor.WriteArtifacts(outDir, artifacts, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range artifacts {
		p := filepath.Join(outDir, a.Name)
		content, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("file not created: %s: %v", a.Name, err)
		}
		if !bytes.Equal(content, a.Content) {
			t.Errorf("%s: got %q, want %q", a.Name, content, a.Content)
		}
	}
}

func TestWriteArtifacts_DryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "sales.qvw")
	artifacts := []extractor.Artifact{
		{Name: "script.qvs", Content: []byte("///")},
	}
	if err := extractor.WriteArtifacts(outDir, artifacts, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Error("expected output dir NOT to exist in dry-run mode")
	}
}

func TestWriteArtifacts_CreatesIntermediateDirs(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "a", "b", "sales.qvw")
	artifacts := []extractor.Artifact{
		{Name: "script.qvs", Content: []byte("///")},
	}
	if err := extractor.WriteArtifacts(outDir, artifacts, false); err != nil {
		t.Fatalf("expected dirs to be auto-created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "script.qvs")); err != nil {
		t.Errorf("file not found: %v", err)
	}
}

func TestWriteArtifacts_FailFast(t *testing.T) {
	// Make the output dir a file so MkdirAll fails.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	_ = os.WriteFile(blocker, []byte("x"), 0644)
	artifacts := []extractor.Artifact{
		{Name: "script.qvs", Content: []byte("///")},
	}
	err := extractor.WriteArtifacts(blocker, artifacts, false)
	if err == nil {
		t.Error("expected error when outDir is a file, got nil")
	}
}
