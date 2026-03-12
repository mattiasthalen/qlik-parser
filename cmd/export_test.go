package cmd_test

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattiasthalen/qlik-script-extractor/cmd"
)

var update = flag.Bool("update", false, "Update golden files")

func TestExportCmd_HelpRegistered(t *testing.T) {
	root := cmd.NewRootCmd()
	root.SetArgs([]string{"export", "--help"})
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("--source")) {
		t.Errorf("expected --source flag in export help, got: %s", out)
	}
}

func TestExportCmd_SourceNotFound(t *testing.T) {
	root := cmd.NewRootCmd()
	root.SetArgs([]string{"export", "--source", "/nonexistent/path/xyz"})
	errBuf := &bytes.Buffer{}
	root.SetErr(errBuf)
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent source, got nil")
	}
	var exitErr *cmd.ExitCodeError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Errorf("expected ExitCodeError(1), got: %v", err)
	}
}

func TestExportCmd_SourceIsFile(t *testing.T) {
	f, _ := os.CreateTemp("", "*.qvw")
	_ = f.Close()
	defer func() { _ = os.Remove(f.Name()) }()

	root := cmd.NewRootCmd()
	root.SetArgs([]string{"export", "--source", f.Name()})
	errBuf := &bytes.Buffer{}
	root.SetErr(errBuf)
	err := root.Execute()
	var exitErr *cmd.ExitCodeError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Errorf("expected ExitCodeError(1) for file source, got: %v", err)
	}
}

func TestExportCmd_EmptySourceDir(t *testing.T) {
	dir := t.TempDir()
	buf := &bytes.Buffer{}
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"export", "--source", dir})
	err := root.Execute()
	if err != nil {
		t.Fatalf("expected no error for empty dir, got: %v", err)
	}
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("0")) {
		t.Errorf("expected 0 in summary for empty dir, got: %q", out)
	}
}

func TestExportCmd_BadFlag_ExitCode2(t *testing.T) {
	root := cmd.NewRootCmd()
	root.SetArgs([]string{"export", "--bogus-unknown-flag"})
	errBuf := &bytes.Buffer{}
	root.SetErr(errBuf)
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
	var exitErr *cmd.ExitCodeError
	if errors.As(err, &exitErr) {
		t.Errorf("expected cobra error (not ExitCodeError) for bad flag, got ExitCodeError(%d)", exitErr.Code)
	}
}

func TestExportCmd_DryRunNoFilesWritten(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	fixturesDir := "../internal/extractor/testdata/fixtures"
	validQVW, err := os.ReadFile(filepath.Join(fixturesDir, "valid.qvw"))
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	_ = os.WriteFile(filepath.Join(srcDir, "test.qvw"), validQVW, 0644)

	root := cmd.NewRootCmd()
	root.SetArgs([]string{"export", "--source", srcDir, "--out", outDir, "--dry-run"})
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, _ := os.ReadDir(outDir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".qvs" {
			t.Errorf("expected no .qvs in dry-run, found: %s", e.Name())
		}
	}
}

func TestExportCmd_Integration_ValidFixture(t *testing.T) {
	fixturesDir := filepath.Join("..", "internal", "extractor", "testdata", "fixtures")
	outDir := t.TempDir()

	root := cmd.NewRootCmd()
	root.SetArgs([]string{
		"export",
		"--source", fixturesDir,
		"--out", outDir,
	})
	buf := &bytes.Buffer{}
	root.SetOut(buf)

	// Some fixtures will error (invalid_zlib, too_short) — that's expected
	_ = root.Execute()

	gotBytes, readErr := os.ReadFile(filepath.Join(outDir, "valid.qvs"))
	if readErr != nil {
		t.Fatalf("expected valid.qvs to be written: %v", readErr)
	}

	goldenPath := filepath.Join(fixturesDir, "valid.qvs.golden")
	if *update {
		_ = os.WriteFile(goldenPath, gotBytes, 0644)
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden file not found: %v — run with -update to create it", err)
	}
	if !bytes.Equal(gotBytes, wantBytes) {
		t.Errorf("output does not match golden file.\ngot:  %q\nwant: %q", gotBytes, wantBytes)
	}
}

func TestExportCmd_Integration_NoScriptIsWarn(t *testing.T) {
	fixturesDir := filepath.Join("..", "internal", "extractor", "testdata", "fixtures")
	outDir := t.TempDir()

	root := cmd.NewRootCmd()
	root.SetArgs([]string{
		"export",
		"--source", fixturesDir,
		"--out", outDir,
	})
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	_ = root.Execute()

	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("no script found")) {
		t.Errorf("expected 'no script found' warn in output, got: %s", out)
	}
}

func TestExportCmd_Integration_ErrorFilesSetExitCode(t *testing.T) {
	fixturesDir := filepath.Join("..", "internal", "extractor", "testdata", "fixtures")
	outDir := t.TempDir()

	root := cmd.NewRootCmd()
	root.SetArgs([]string{
		"export",
		"--source", fixturesDir,
		"--out", outDir,
	})
	err := root.Execute()

	var exitErr *cmd.ExitCodeError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Errorf("expected ExitCodeError(1) due to corrupt fixtures, got: %v", err)
	}
}
