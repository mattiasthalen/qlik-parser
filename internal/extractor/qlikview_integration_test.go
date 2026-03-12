package extractor_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattiasthalen/qlik-parser/cmd"
	"github.com/mattiasthalen/qlik-parser/internal/extractor"
)

const qlikviewTestdata = "testdata/qlikview"

func skipIfNoQlikviewFixtures(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(qlikviewTestdata); os.IsNotExist(err) {
		t.Skip("real QVW fixtures not present (gitignored) — skipping")
	}
}

func TestQlikview_WalkerFindsAll34Files(t *testing.T) {
	skipIfNoQlikviewFixtures(t)

	paths, warns := extractor.Walk(qlikviewTestdata)

	if len(warns) != 0 {
		t.Errorf("expected no warnings, got: %v", warns)
	}
	if len(paths) != 18 {
		t.Errorf("expected 18 QVW files, got %d: %v", len(paths), paths)
	}
}

func TestQlikview_AllFilesExtractWithoutError(t *testing.T) {
	skipIfNoQlikviewFixtures(t)

	paths, _ := extractor.Walk(qlikviewTestdata)
	for _, p := range paths {
		rel, _ := filepath.Rel(qlikviewTestdata, p)
		t.Run(rel, func(t *testing.T) {
			_, err := extractor.ExtractScript(p)
			if err != nil {
				var noScript *extractor.NoScriptError
				if errors.As(err, &noScript) {
					t.Errorf("no script found in %s", rel)
				} else {
					t.Errorf("extraction error for %s: %v", rel, err)
				}
			}
		})
	}
}

func TestQlikview_AllScriptsStartWithTripleSlash(t *testing.T) {
	skipIfNoQlikviewFixtures(t)

	paths, _ := extractor.Walk(qlikviewTestdata)
	for _, p := range paths {
		rel, _ := filepath.Rel(qlikviewTestdata, p)
		script, err := extractor.ExtractScript(p)
		if err != nil {
			continue // covered by TestQlikview_AllFilesExtractWithoutError
		}
		if !strings.HasPrefix(script, "///") {
			t.Errorf("%s: expected script to start with ///, got: %q", rel, script[:min(30, len(script))])
		}
	}
}

func TestQlikview_ExportMirrorMode_PreservesSubdirStructure(t *testing.T) {
	skipIfNoQlikviewFixtures(t)

	srcDir, _ := filepath.Abs(qlikviewTestdata)
	outDir := t.TempDir()

	root := cmd.NewRootCmd()
	root.SetArgs([]string{"export", "--source", srcDir, "--out", outDir})
	buf := &bytes.Buffer{}
	root.SetOut(buf)

	if err := root.Execute(); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	// Verify one file from each subdir to confirm structure is mirrored
	expected := []string{
		filepath.Join(outDir, "extract", "QVD Extract IFS.qvs"),
		filepath.Join(outDir, "load", "IFS Recipe Structure.qvs"),
		filepath.Join(outDir, "transform", "QVD Transform IFS.qvs"),
	}
	for _, f := range expected {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("expected mirrored output file not found: %s", f)
		}
	}
}

func TestQlikview_ExportDryRun_WritesNoFiles(t *testing.T) {
	skipIfNoQlikviewFixtures(t)

	srcDir, _ := filepath.Abs(qlikviewTestdata)
	outDir := t.TempDir()

	root := cmd.NewRootCmd()
	root.SetArgs([]string{"export", "--source", srcDir, "--out", outDir, "--dry-run"})
	buf := &bytes.Buffer{}
	root.SetOut(buf)

	if err := root.Execute(); err != nil {
		t.Fatalf("dry-run export failed: %v", err)
	}

	entries, _ := os.ReadDir(outDir)
	if len(entries) != 0 {
		t.Errorf("dry-run wrote %d files/dirs, expected none", len(entries))
	}
}

func TestQlikview_ExportSucceeds_ExitCode0(t *testing.T) {
	skipIfNoQlikviewFixtures(t)

	srcDir, _ := filepath.Abs(qlikviewTestdata)
	outDir := t.TempDir()

	root := cmd.NewRootCmd()
	root.SetArgs([]string{"export", "--source", srcDir, "--out", outDir})
	buf := &bytes.Buffer{}
	root.SetOut(buf)

	err := root.Execute()
	if err != nil {
		t.Errorf("expected exit 0 (nil error) for all-valid QVW files, got: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Extracted 18 scripts") {
		t.Errorf("expected 'Extracted 18 scripts' in summary, got: %q", out)
	}
}
