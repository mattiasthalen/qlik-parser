package cmd_test

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattiasthalen/qlik-parser/cmd"
)

var update = flag.Bool("update", false, "Update golden files")

func TestExtractCmd_HelpRegistered(t *testing.T) {
	root := cmd.NewRootCmd()
	root.SetArgs([]string{"extract", "--help"})
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("--source")) {
		t.Errorf("expected --source flag in extract help, got: %s", out)
	}
}

func TestExtractCmd_SourceNotFound(t *testing.T) {
	root := cmd.NewRootCmd()
	root.SetArgs([]string{"extract", "--source", "/nonexistent/path/xyz"})
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

func TestExtractCmd_SourceIsFile(t *testing.T) {
	f, _ := os.CreateTemp("", "*.qvw")
	_ = f.Close()
	defer func() { _ = os.Remove(f.Name()) }()

	root := cmd.NewRootCmd()
	root.SetArgs([]string{"extract", "--source", f.Name()})
	errBuf := &bytes.Buffer{}
	root.SetErr(errBuf)
	err := root.Execute()
	var exitErr *cmd.ExitCodeError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Errorf("expected ExitCodeError(1) for file source, got: %v", err)
	}
}

func TestExtractCmd_EmptySourceDir(t *testing.T) {
	dir := t.TempDir()
	buf := &bytes.Buffer{}
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"extract", "--source", dir})
	err := root.Execute()
	if err != nil {
		t.Fatalf("expected no error for empty dir, got: %v", err)
	}
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("0")) {
		t.Errorf("expected 0 in summary for empty dir, got: %q", out)
	}
}

func TestExtractCmd_BadFlag_ExitCode2(t *testing.T) {
	root := cmd.NewRootCmd()
	root.SetArgs([]string{"extract", "--bogus-unknown-flag"})
	errBuf := &bytes.Buffer{}
	root.SetErr(errBuf)
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
	var exitErr *cmd.ExitCodeError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitCodeError(2) for bad flag, got: %v", err)
	}
}

func TestExtractCmd_DryRunNoFilesWritten(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	fixturesDir := "../internal/extractor/testdata/fixtures"
	validQVW, err := os.ReadFile(filepath.Join(fixturesDir, "valid.qvw"))
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	_ = os.WriteFile(filepath.Join(srcDir, "test.qvw"), validQVW, 0644)

	root := cmd.NewRootCmd()
	root.SetArgs([]string{"extract", "--source", srcDir, "--out", outDir, "--dry-run"})
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, _ := os.ReadDir(outDir)
	if len(entries) != 0 {
		t.Errorf("expected no output in dry-run, found %d entries", len(entries))
	}
}

func TestExtractCmd_Integration_ValidFixture(t *testing.T) {
	fixturesDir := filepath.Join("..", "internal", "extractor", "testdata", "fixtures")
	outDir := t.TempDir()

	root := cmd.NewRootCmd()
	root.SetArgs([]string{
		"extract",
		"--source", fixturesDir,
		"--out", outDir,
	})
	buf := &bytes.Buffer{}
	root.SetOut(buf)

	// Some fixtures will error (invalid_zlib, too_short) — that's expected
	_ = root.Execute()

	gotBytes, readErr := os.ReadFile(filepath.Join(outDir, "valid.qvw", "script.qvs"))
	if readErr != nil {
		t.Fatalf("expected valid.qvw/script.qvs to be written: %v", readErr)
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
	normalize := func(b []byte) string { return strings.ReplaceAll(string(b), "\r\n", "\n") }
	if normalize(gotBytes) != normalize(wantBytes) {
		t.Errorf("output does not match golden file.\ngot:  %q\nwant: %q", gotBytes, wantBytes)
	}
}

func TestExtractCmd_Integration_NoScriptIsWarn(t *testing.T) {
	fixturesDir := filepath.Join("..", "internal", "extractor", "testdata", "fixtures")
	outDir := t.TempDir()

	root := cmd.NewRootCmd()
	root.SetArgs([]string{
		"extract",
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

func TestExtractCmd_ErrorMessage_DoesNotContainAbsPath(t *testing.T) {
	srcDir := t.TempDir()
	// Write a too-short QVW file (< 23 bytes) which triggers an error with path in message
	_ = os.WriteFile(filepath.Join(srcDir, "short.qvw"), []byte("tooshort"), 0644)

	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	root := cmd.NewRootCmd()
	root.SetArgs([]string{"extract", "--source", srcDir})
	root.SetOut(buf)
	root.SetErr(errBuf)
	_ = root.Execute()

	out := buf.String()
	if strings.Contains(out, srcDir) {
		t.Errorf("error output should not contain abs path %q, got: %s", srcDir, out)
	}
}

func TestExtractCmd_Integration_ErrorFilesSetExitCode(t *testing.T) {
	fixturesDir := filepath.Join("..", "internal", "extractor", "testdata", "fixtures")
	outDir := t.TempDir()

	root := cmd.NewRootCmd()
	root.SetArgs([]string{
		"extract",
		"--source", fixturesDir,
		"--out", outDir,
	})
	err := root.Execute()

	var exitErr *cmd.ExitCodeError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Errorf("expected ExitCodeError(1) due to corrupt fixtures, got: %v", err)
	}
}

func TestExtractCmd_NoArtifactSelected(t *testing.T) {
	root := cmd.NewRootCmd()
	root.SetArgs([]string{"extract", "--script=false", "--source", "/nonexistent/path/xyz"})
	errBuf := &bytes.Buffer{}
	root.SetErr(errBuf)
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no artifact selected, got nil")
	}
	var exitErr *cmd.ExitCodeError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Errorf("expected ExitCodeError(1), got: %v", err)
	}
	if !bytes.Contains(errBuf.Bytes(), []byte("no artifact type selected")) {
		t.Errorf("expected 'no artifact type selected' in stderr, got: %s", errBuf.String())
	}
}
