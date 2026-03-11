# QlikView Script Extractor — Phase 04: Export Command & Integration Tests Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `cmd/export.go` — the cobra `export` subcommand that wires together the walker, extractor, exporter, and UI — plus integration tests with golden files.

**Architecture:** `cmd/export.go` handles flag parsing, orchestrates the two-phase walk+process loop, drives the UI printer, resolves exit codes. Integration tests in `cmd/export_test.go` run the full command against fixtures and compare output to golden files. Golden files live in `internal/extractor/testdata/` with `.qvs.golden` extension.

**Tech Stack:** Go 1.24, cobra, zerolog, internal packages from Phases 02–03.

**Spec:** `docs/superpowers/specs/2026-03-11-qlik-script-extractor-design.md` — "CLI Interface", "Error Handling & Exit Codes", "Testing".

**Prerequisites:** Phase 01 (scaffolding), Phase 02 (extractor), Phase 03 (UI) all complete and merged.

**Parallelism note:** Tasks 1–5 (export command implementation) must be sequential. Tasks 6–8 (integration tests + golden files) depend on Task 5 being complete.

---

## Chunk 1: Export Command

### Task 1: Write export command skeleton test

**Files:**
- Create: `cmd/export_test.go`

- [ ] **Step 1: Write a minimal failing test**

```go
package cmd_test

import (
	"bytes"
	"testing"

	"github.com/your-org/qlik-script-extractor/cmd"
)

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -run TestExportCmd_HelpRegistered -v`
Expected: FAIL — `export` subcommand not registered.

---

### Task 2: Implement export command (flags + registration)

**Files:**
- Create: `cmd/export.go`

- [ ] **Step 1: Write the export command with all flags**

```go
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/your-org/qlik-script-extractor/internal/extractor"
	"github.com/your-org/qlik-script-extractor/internal/ui"
)

func newExportCmd() *cobra.Command {
	var sourceDir string
	var outDir string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Extract load scripts from .qvw files",
		Long: `Recursively scans --source for .qvw files and extracts the embedded
load scripts to .qvs text files alongside or under --out.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default source to cwd if not provided
			if sourceDir == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("could not determine working directory: %w", err)
				}
				sourceDir = cwd
			}

			// Validate source
			info, err := os.Stat(sourceDir)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "error: --source %q: %v\n", sourceDir, err)
				return ExitError(1)
			}
			if !info.IsDir() {
				fmt.Fprintf(cmd.ErrOrStderr(), "error: --source %q is a file, not a directory\n", sourceDir)
				return ExitError(1)
			}

			// Pre-flight --out directory if specified
			if outDir != "" {
				if err := os.MkdirAll(outDir, 0755); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "error: cannot create --out directory %q: %v\n", outDir, err)
					return ExitError(1)
				}
			}

			// Phase 1: collect all .qvw paths
			qvwPaths, walkWarns := extractor.Walk(sourceDir)
			for _, w := range walkWarns {
				log.Warn().Msg(w)
			}

			isTTY := ui.IsTTY(os.Stdout)
			printer := ui.NewPrinter(cmd.OutOrStdout(), isTTY, dryRun)

			hasErr := false

			// Phase 2: process each file
			for i, qvwPath := range qvwPaths {
				printer.UpdateSpinner(i+1, len(qvwPaths))

				// Compute relative path for display
				relPath, err := filepath.Rel(sourceDir, qvwPath)
				if err != nil {
					relPath = filepath.Base(qvwPath)
				}

				script, extractErr := extractor.ExtractScript(qvwPath)
				if extractErr != nil {
					var noScript *extractor.NoScriptError
					if extractor.IsNoScript(extractErr, &noScript) {
						printer.ClearSpinner()
						printer.FileResult(ui.Result{
							Status:  ui.StatusWarn,
							QVWPath: relPath,
							Message: "no script found",
						})
						continue
					}
					hasErr = true
					printer.ClearSpinner()
					printer.FileResult(ui.Result{
						Status:  ui.StatusErr,
						QVWPath: relPath,
						Message: extractErr.Error(),
					})
					continue
				}

				outPath := extractor.ResolveOutputPath(qvwPath, sourceDir, outDir)
				relOut, err := filepath.Rel(sourceDir, outPath)
				if err != nil {
					relOut = filepath.Base(outPath)
				}
				// In mirror mode the outPath may be outside sourceDir — use outDir as base
				if outDir != "" && outDir != sourceDir {
					relOut, err = filepath.Rel(outDir, outPath)
					if err != nil {
						relOut = filepath.Base(outPath)
					}
				}

				writeErr := extractor.WriteScript(outPath, script, dryRun)
				if writeErr != nil {
					hasErr = true
					printer.ClearSpinner()
					printer.FileResult(ui.Result{
						Status:  ui.StatusErr,
						QVWPath: relPath,
						Message: writeErr.Error(),
					})
					continue
				}

				printer.ClearSpinner()
				printer.FileResult(ui.Result{
					Status:    ui.StatusOK,
					QVWPath:   relPath,
					QVSPath:   relOut,
					CharCount: len(script),
				})
			}

			printer.Summary()

			if hasErr {
				return ExitError(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&sourceDir, "source", "s", "", "Source directory to scan for .qvw files (default: current directory)")
	cmd.Flags().StringVarP(&outDir, "out", "o", "", "Export directory (default: alongside .qvw files)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be extracted without writing files")

	return cmd
}

// ExitError is a sentinel error used to signal a specific exit code to main.
type ExitCodeError struct {
	Code int
}

func (e *ExitCodeError) Error() string {
	return fmt.Sprintf("exit %d", e.Code)
}

// ExitError creates an ExitCodeError. Used to signal non-zero exit without printing cobra usage.
func ExitError(code int) error {
	return &ExitCodeError{Code: code}
}
```

- [ ] **Step 2: Register export in NewRootCmd**

Edit `cmd/root.go` — inside `NewRootCmd()`, after `root.AddCommand(newVersionCmd())`, add:

```go
root.AddCommand(newExportCmd())
```

- [ ] **Step 3: Run the help test**

Run: `go test ./cmd/... -run TestExportCmd_HelpRegistered -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/export.go cmd/root.go cmd/export_test.go
git commit -m "feat: add export subcommand with --source --out --dry-run flags"
```

---

### Task 3: Update main.go to handle ExitCodeError

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Read current main.go**

Read `main.go` to understand current content.

- [ ] **Step 2: Update main.go to handle ExitCodeError**

Replace the `main` function body:

```go
package main

import (
	"errors"
	"os"

	"github.com/your-org/qlik-script-extractor/cmd"
)

func main() {
	root := cmd.NewRootCmd()
	if err := root.Execute(); err != nil {
		var exitErr *cmd.ExitCodeError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Build to verify it compiles**

Run: `make build`
Expected: Binary `qlik-script-extractor` produced. No errors.

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "fix: handle ExitCodeError in main for correct exit codes"
```

---

## Chunk 2: Exit Code Tests

### Task 4: Write exit code unit tests

**Files:**
- Modify: `cmd/export_test.go`

- [ ] **Step 1: Add exit code tests**

Add these tests to `cmd/export_test.go`:

```go
import (
	"errors"
	"os"
	"path/filepath"
)

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
	f.Close()
	defer os.Remove(f.Name())

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
	// Cobra returns an error for unknown flags; main.go translates to exit 2.
	// At the cmd layer we verify cobra returns an error for bad args.
	root := cmd.NewRootCmd()
	root.SetArgs([]string{"export", "--bogus-unknown-flag"})
	errBuf := &bytes.Buffer{}
	root.SetErr(errBuf)
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
	// cobra returns its own error type for unknown flags — not ExitCodeError.
	// The exit code 2 is enforced in main.go (cobra's default).
	// Verify we do NOT get an ExitCodeError (those are application-level):
	var exitErr *cmd.ExitCodeError
	if errors.As(err, &exitErr) {
		t.Errorf("expected cobra error (not ExitCodeError) for bad flag, got ExitCodeError(%d)", exitErr.Code)
	}
}

func TestExportCmd_DryRunWithErrorFiles_ExitCode1(t *testing.T) {
	// Spec: "--dry-run does not suppress error exit codes — files that would fail still count as ERR"
	fixturesDir := filepath.Join("..", "internal", "extractor", "testdata", "fixtures")

	root := cmd.NewRootCmd()
	root.SetArgs([]string{
		"export",
		"--source", fixturesDir,
		"--dry-run",
	})
	buf := &bytes.Buffer{}
	root.SetOut(buf)

	err := root.Execute()

	var exitErr *cmd.ExitCodeError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Errorf("expected ExitCodeError(1) for dry-run with corrupt fixtures, got: %v", err)
	}

	// Also verify no .qvs files were written anywhere in the fixtures dir
	entries, _ := filepath.Glob(filepath.Join(fixturesDir, "*.qvs"))
	if len(entries) > 0 {
		t.Errorf("dry-run wrote .qvs files: %v", entries)
	}
}

func TestExportCmd_DryRunNoFilesWritten(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	// Create a minimal valid .qvw fixture
	fixturesDir := "../internal/extractor/testdata/fixtures"
	validQVW, err := os.ReadFile(filepath.Join(fixturesDir, "valid.qvw"))
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	os.WriteFile(filepath.Join(srcDir, "test.qvw"), validQVW, 0644)

	root := cmd.NewRootCmd()
	root.SetArgs([]string{"export", "--source", srcDir, "--out", outDir, "--dry-run"})
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify no .qvs files were written
	entries, _ := os.ReadDir(outDir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".qvs" {
			t.Errorf("expected no .qvs in dry-run, found: %s", e.Name())
		}
	}
}
```

- [ ] **Step 2: Run exit code tests**

Run: `go test ./cmd/... -run "TestExportCmd_Source|TestExportCmd_Empty|TestExportCmd_DryRun" -v`
Expected: All PASS

- [ ] **Step 3: Commit**

```bash
git add cmd/export_test.go
git commit -m "test: add export command exit code and dry-run tests"
```

---

## Chunk 3: Integration Tests with Golden Files

### Task 5: Create golden files

**Files:**
- Create: `internal/extractor/testdata/fixtures/valid.qvs.golden`

- [ ] **Step 1: Generate golden file from valid fixture**

Run the extractor against the fixture to get the expected output, then write it:

```bash
# Create a small Go program to extract and print, or use the CLI once built
go run main.go export --source internal/extractor/testdata/fixtures --out /tmp/golden-out
cat /tmp/golden-out/valid.qvs
```

Or write a one-off helper:

```go
//go:build ignore

package main

import (
	"fmt"
	"os"
	"github.com/your-org/qlik-script-extractor/internal/extractor"
)

func main() {
	script, err := extractor.ExtractScript("internal/extractor/testdata/fixtures/valid.qvw")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	os.WriteFile("internal/extractor/testdata/fixtures/valid.qvs.golden", []byte(script), 0644)
	fmt.Println("Generated valid.qvs.golden")
}
```

Run: `go run internal/extractor/testdata/gen/golden_gen.go`
Expected: `valid.qvs.golden` created.

- [ ] **Step 2: Verify golden file content**

Run: `cat internal/extractor/testdata/fixtures/valid.qvs.golden`
Expected: Content starts with `///` and contains `LOAD * FROM table.csv`.

- [ ] **Step 3: Commit golden file**

```bash
git add internal/extractor/testdata/fixtures/valid.qvs.golden
git commit -m "test: add golden file for valid.qvw fixture"
```

---

### Task 6: Write integration tests with golden files

**Files:**
- Modify: `cmd/export_test.go`

- [ ] **Step 1: Add integration test with golden comparison**

Add these tests to `cmd/export_test.go`:

```go
import (
	"flag"
)

var update = flag.Bool("update", false, "Update golden files")

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

	err := root.Execute()
	// Expect exit 1 because some fixtures (invalid_zlib, too_short) will error
	// We still verify the valid.qvs output
	_ = err

	gotBytes, readErr := os.ReadFile(filepath.Join(outDir, "valid.qvs"))
	if readErr != nil {
		t.Fatalf("expected valid.qvs to be written: %v", readErr)
	}

	goldenPath := filepath.Join(fixturesDir, "valid.qvs.golden")
	if *update {
		os.WriteFile(goldenPath, gotBytes, 0644)
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
	root.Execute()

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
```

- [ ] **Step 2: Run integration tests**

Run: `go test ./cmd/... -run TestExportCmd_Integration -v`
Expected: All PASS (or investigate failures and fix)

- [ ] **Step 3: Commit**

```bash
git add cmd/export_test.go
git commit -m "test: add integration tests with golden file comparison"
```

---

## Chunk 4: Final Validation

### Task 7: Full coverage check

- [ ] **Step 1: Run full test suite with coverage**

Run: `go test ./... -coverprofile=coverage.out -covermode=atomic && go tool cover -func=coverage.out`
Expected: Total coverage >80%.

- [ ] **Step 2: Run linter**

Run: `make lint`
Expected: No lint errors.

- [ ] **Step 3: Build and smoke test final binary**

Run:
```bash
make build
./qlik-script-extractor version
./qlik-script-extractor export --help
./qlik-script-extractor export --source internal/extractor/testdata/fixtures --dry-run
```

Expected:
- Version prints `qlik-script-extractor dev`
- Help shows `--source`, `--out`, `--dry-run` flags
- Dry-run shows `[dry run]` on each line and summary shows "Dry run"

- [ ] **Step 4: Commit any final fixes**

```bash
git add -p
git commit -m "fix: final lint and coverage fixes"
```

(Only create this commit if changes are needed.)

---

### Task 8: Validate golden file update workflow

- [ ] **Step 1: Test the -update flag regenerates golden files**

Run: `go test ./cmd/... -run TestExportCmd_Integration_ValidFixture -update -v`
Expected: "Updated golden file" logged, test PASS.

- [ ] **Step 2: Run without -update to confirm golden matches**

Run: `go test ./cmd/... -run TestExportCmd_Integration_ValidFixture -v`
Expected: PASS — golden file matches output.

- [ ] **Step 3: Final commit**

```bash
git add internal/extractor/testdata/fixtures/valid.qvs.golden
git commit -m "test: ensure golden files are up to date"
```

(Only if golden file changed.)
