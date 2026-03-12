# QVF Script Extraction Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add support for extracting load scripts from Qlik Sense `.qvf` files alongside the existing `.qvw` support.

**Architecture:** A new `ExtractScriptFromQVF` function scans the binary file for zlib-compressed blocks, decompresses each candidate, and JSON-unmarshals to find the `qScript` field. The walker is extended to collect both extensions; the CLI dispatches by extension. Output filenames use `.qvf.qvs` to avoid collision with same-named `.qvw` outputs.

**Tech Stack:** Go stdlib (`compress/zlib`, `encoding/json`, `path/filepath`), existing `NoScriptError` type, Cobra CLI, testify-free table-driven tests (project uses stdlib `testing` only).

**Worktree:** `/workspaces/qlik-parser/.worktrees/feat/qvf-extraction`
**Run all tests:** `go test ./...` from the worktree root.
**Spec:** `docs/superpowers/specs/2026-03-12-qvf-extraction-design.md`

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `internal/extractor/qvf.go` | `ExtractScriptFromQVF` — zlib scan + JSON extraction |
| Create | `internal/extractor/qvf_test.go` | Unit tests for QVF extractor |
| Modify | `internal/extractor/walker.go` | Collect `.qvf` in addition to `.qvw` |
| Modify | `internal/extractor/walker_test.go` | Rename ignore test, add QVF collection test |
| Modify | `internal/extractor/exporter.go` | Generalize `ResolveOutputPath` for `.qvf` → `.qvf.qvs` |
| Modify | `internal/extractor/exporter_test.go` | Add `.qvf` output path cases |
| Modify | `internal/extractor/qvw_test.go` | Tighten prefix assertions from `"///"` to `"///$tab"` |
| Modify | `internal/ui/output.go` | Rename `Result.QVWPath` → `Result.SrcPath` |
| Modify | `internal/ui/output_test.go` | Update all `QVWPath:` literals to `SrcPath:` |
| Modify | `cmd/extract.go` | Dispatch by extension; update help text; update `Result` literals |
| Modify | `internal/extractor/qlikview_integration_test.go` | Update counts, dispatch, assertions |

---

## Chunk 1: QVF Extractor

### Task 1: Rename `Result.QVWPath` → `Result.SrcPath`

This is a pure rename with no logic change. Do it first so all subsequent tasks use the correct field name.

**Files:**
- Modify: `internal/ui/output.go:22-27`
- Modify: `internal/ui/output_test.go` (all `QVWPath:` literals)
- Modify: `cmd/extract.go` (all `QVWPath:` literals)

- [ ] **Step 1: Rename the field in `output.go`**

In `internal/ui/output.go`, change line 23 from `QVWPath string` to `SrcPath string`.
Also update the three method bodies that reference `r.QVWPath` (lines 71, 80, 89) to `r.SrcPath`.

- [ ] **Step 2: Update `output_test.go`**

Replace every `QVWPath:` with `SrcPath:` in `internal/ui/output_test.go`. There are 10 occurrences (lines 20, 41, 57, 74, 88, 89, 90, 91, 102, 103).

- [ ] **Step 3: Update `cmd/extract.go`**

Replace every `QVWPath:` with `SrcPath:` in `cmd/extract.go`. There are 4 occurrences (lines 85, 98, 121, 130). Note: `QVSPath:` on the nearby line must NOT be renamed.

- [ ] **Step 4: Verify tests pass**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/output.go internal/ui/output_test.go cmd/extract.go
git commit -m "refactor: rename Result.QVWPath to SrcPath"
git push
```

---

### Task 2: Implement `ExtractScriptFromQVF` with TDD

**Files:**
- Create: `internal/extractor/qvf.go`
- Create: `internal/extractor/qvf_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/extractor/qvf_test.go`:

```go
package extractor_test

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/mattiasthalen/qlik-parser/internal/extractor"
)

// buildQVFFixture builds a fake QVF file containing one zlib stream
// that decompresses to the given JSON bytes, preceded by arbitrary junk.
func buildQVFFixture(t *testing.T, jsonPayload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, _ = w.Write(jsonPayload)
	_ = w.Close()

	// Prepend 64 bytes of junk (0xFF) so the scanner has to find the stream.
	junk := make([]byte, 64)
	for i := range junk {
		junk[i] = 0xFF
	}
	return append(junk, buf.Bytes()...)
}

func TestExtractScriptFromQVF_ValidScript(t *testing.T) {
	script := "///$tab Main\r\nSET ThousandSep=',';"
	payload, _ := json.Marshal(map[string]string{"qScript": script})
	data := buildQVFFixture(t, payload)

	f, err := os.CreateTemp("", "valid_*.qvf")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	_, _ = f.Write(data)
	_ = f.Close()

	got, err := extractor.ExtractScriptFromQVF(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != script {
		t.Errorf("expected %q, got %q", script, got)
	}
}

func TestExtractScriptFromQVF_NoQScript(t *testing.T) {
	// A valid zlib stream with JSON that has no qScript field.
	payload, _ := json.Marshal(map[string]string{"other": "value"})
	data := buildQVFFixture(t, payload)

	f, err := os.CreateTemp("", "noscript_*.qvf")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	_, _ = f.Write(data)
	_ = f.Close()

	_, err = extractor.ExtractScriptFromQVF(f.Name())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var noScript *extractor.NoScriptError
	if !errors.As(err, &noScript) {
		t.Errorf("expected *NoScriptError, got %T: %v", err, err)
	}
}

func TestExtractScriptFromQVF_FileNotFound(t *testing.T) {
	_, err := extractor.ExtractScriptFromQVF("/nonexistent/path.qvf")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if errors.As(err, new(*extractor.NoScriptError)) {
		t.Errorf("expected os-level error, got NoScriptError")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/extractor/... -run TestExtractScriptFromQVF -v
```

Expected: compilation error — `extractor.ExtractScriptFromQVF` does not exist yet.

- [ ] **Step 3: Implement `qvf.go`**

Create `internal/extractor/qvf.go`:

```go
package extractor

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// qvfPayload is used to unmarshal only the qScript field from a QVF JSON block.
type qvfPayload struct {
	QScript string `json:"qScript"`
}

// ExtractScriptFromQVF reads a .qvf file and returns the embedded load script.
//
// A .qvf file is a proprietary binary container holding multiple zlib-compressed
// blocks. This function scans the file for zlib stream candidates (CMF byte 0x78
// followed by a valid FLG byte), decompresses each, and JSON-unmarshals to find
// the block containing a non-empty "qScript" field.
//
// Errors:
//   - os read error if the file cannot be read
//   - *NoScriptError if no block with a qScript field is found
func ExtractScriptFromQVF(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}

	// Valid zlib FLG bytes for CMF=0x78 (deflate, window size 32KB):
	// The pair (CMF*256+FLG) must be divisible by 31.
	validFLG := map[byte]bool{0x01: true, 0x5E: true, 0x9C: true, 0xDA: true}

	for i := 0; i < len(data)-1; i++ {
		if data[i] != 0x78 || !validFLG[data[i+1]] {
			continue
		}
		r, err := zlib.NewReader(bytes.NewReader(data[i:]))
		if err != nil {
			continue
		}
		decompressed, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			continue
		}
		var payload qvfPayload
		if err := json.Unmarshal(decompressed, &payload); err != nil {
			continue
		}
		if payload.QScript != "" {
			return payload.QScript, nil
		}
	}

	return "", &NoScriptError{Path: path}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/extractor/... -run TestExtractScriptFromQVF -v
```

Expected: all three tests PASS.

- [ ] **Step 5: Run full test suite**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 6: Commit**

```bash
git add internal/extractor/qvf.go internal/extractor/qvf_test.go
git commit -m "feat: add ExtractScriptFromQVF for Qlik Sense .qvf files"
git push
```

---

## Chunk 2: Walker + Exporter

### Task 3: Extend Walker to Collect `.qvf` Files

**Files:**
- Modify: `internal/extractor/walker.go:29`
- Modify: `internal/extractor/walker_test.go`

- [ ] **Step 1: Update the walker test**

In `internal/extractor/walker_test.go`:

1. Rename `TestWalkIgnoresNonQVW` to `TestWalkIgnoresUnrelatedExtensions` and change its fixture list from `{"a.qvf", "b.txt", "c.qvs", "d.QVW"}` to `{"a.txt", "b.qvs", "c.QVW"}`.

2. Add a new test after `TestWalkFindsQVWFiles`:

```go
func TestWalkFindsQVFFiles(t *testing.T) {
	root := t.TempDir()
	files := []string{
		filepath.Join(root, "app.qvf"),
		filepath.Join(root, "sub", "nested.qvf"),
		filepath.Join(root, "ignore.txt"),
	}
	_ = os.MkdirAll(filepath.Join(root, "sub"), 0755)
	for _, f := range files {
		_ = os.WriteFile(f, []byte{0x00}, 0644)
	}

	got, warns := extractor.Walk(root)

	if len(warns) != 0 {
		t.Errorf("expected no warns, got %v", warns)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 .qvf files, got %d: %v", len(got), got)
	}
}
```

- [ ] **Step 2: Run tests to verify the new test fails and the renamed test passes**

```bash
go test ./internal/extractor/... -run "TestWalkFindsQVFFiles|TestWalkIgnoresUnrelatedExtensions" -v
```

Expected: `TestWalkFindsQVFFiles` FAIL (walker not yet updated), `TestWalkIgnoresUnrelatedExtensions` PASS.

- [ ] **Step 3: Update `walker.go`**

In `internal/extractor/walker.go`, change line 29 from:

```go
if !d.IsDir() && filepath.Ext(path) == ".qvw" {
```

to:

```go
ext := filepath.Ext(path)
if !d.IsDir() && (ext == ".qvw" || ext == ".qvf") {
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/extractor/... -run "TestWalkFindsQVFFiles|TestWalkIgnoresUnrelatedExtensions|TestWalkFindsQVWFiles" -v
```

Expected: all three PASS.

- [ ] **Step 5: Run full test suite**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 6: Commit**

```bash
git add internal/extractor/walker.go internal/extractor/walker_test.go
git commit -m "feat: walk collects .qvf files in addition to .qvw"
git push
```

---

### Task 4: Generalize `ResolveOutputPath` for QVF

**Files:**
- Modify: `internal/extractor/exporter.go`
- Modify: `internal/extractor/exporter_test.go`

- [ ] **Step 1: Add failing tests for `.qvf` output paths**

Add to `internal/extractor/exporter_test.go` (after the existing tests):

```go
func TestResolveOutputPath_QVF_Alongside(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix path test not applicable on Windows")
	}
	got := extractor.ResolveOutputPath("/data/etl/app.qvf", "/data", "")
	want := "/data/etl/app.qvf.qvs"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestResolveOutputPath_QVF_Mirror(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix path test not applicable on Windows")
	}
	got := extractor.ResolveOutputPath("/data/etl/app.qvf", "/data", "/out")
	want := "/out/etl/app.qvf.qvs"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/extractor/... -run "TestResolveOutputPath_QVF" -v
```

Expected: FAIL — `.qvf` input currently produces `.qvf` → strips `.qvw` (no match) so base is `app.qvf` → `app.qvf.qvs` — actually check what happens. If they pass already (wrong reasons), investigate before proceeding.

- [ ] **Step 3: Update `exporter.go`**

Replace `ResolveOutputPath` in `internal/extractor/exporter.go` with:

```go
// ResolveOutputPath computes the destination path for the extracted script.
//
//   - inputPath: absolute path to the source file (.qvw or .qvf)
//   - sourceDir: the --source directory (used to compute relative path)
//   - outDir:    the --out directory; empty string or equal to sourceDir → alongside mode
//
// Output extensions:
//   - .qvw → .qvs
//   - .qvf → .qvf.qvs  (double-extension avoids collision with same-named .qvw output)
func ResolveOutputPath(inputPath, sourceDir, outDir string) string {
	ext := filepath.Ext(inputPath)
	var outExt string
	switch ext {
	case ".qvf":
		outExt = ".qvf.qvs"
	default:
		outExt = ".qvs"
	}
	base := strings.TrimSuffix(filepath.Base(inputPath), ext) + outExt

	if outDir == "" || outDir == sourceDir {
		return filepath.Join(filepath.Dir(inputPath), base)
	}

	rel, err := filepath.Rel(sourceDir, filepath.Dir(inputPath))
	if err != nil {
		return filepath.Join(outDir, base)
	}
	return filepath.Join(outDir, rel, base)
}
```

- [ ] **Step 4: Run tests to verify all pass**

```bash
go test ./internal/extractor/... -run "TestResolveOutputPath" -v
```

Expected: all 5 tests (3 existing + 2 new) PASS.

- [ ] **Step 5: Run full test suite**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 6: Commit**

```bash
git add internal/extractor/exporter.go internal/extractor/exporter_test.go
git commit -m "feat: ResolveOutputPath supports .qvf -> .qvf.qvs output"
git push
```

---

## Chunk 3: CLI + Tests

### Task 5: Wire QVF into the CLI

**Files:**
- Modify: `cmd/extract.go`

- [ ] **Step 1: Update the extract command**

In `cmd/extract.go`, make these changes:

1. Change the `for` loop body to dispatch by extension. Replace:

```go
scriptContent, extractErr := extractor.ExtractScript(qvwPath)
```

with:

```go
var scriptContent string
var extractErr error
switch filepath.Ext(qvwPath) {
case ".qvf":
    scriptContent, extractErr = extractor.ExtractScriptFromQVF(qvwPath)
default:
    scriptContent, extractErr = extractor.ExtractScript(qvwPath)
}
```

2. Update `Use`, `Short`, and `Long` to mention both formats:

```go
Use:   "extract",
Short: "Extract artifacts from .qvw and .qvf files",
Long: `Recursively scans --source for .qvw and .qvf files and extracts embedded
artifacts to text files alongside or under --out.`,
```

3. Update the `--source` flag description:

```go
cmd.Flags().StringVarP(&sourceDir, "source", "s", "", "Source directory to scan for .qvw and .qvf files (default: current directory)")
```

- [ ] **Step 2: Run full test suite**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 3: Commit**

```bash
git add cmd/extract.go
git commit -m "feat: CLI dispatches .qvf to ExtractScriptFromQVF"
git push
```

---

### Task 6: Tighten `///$tab` Assertions in QVW Unit Tests

**Files:**
- Modify: `internal/extractor/qvw_test.go:21,65`

- [ ] **Step 1: Update the assertions**

In `internal/extractor/qvw_test.go`:

- Line 21: change `strings.HasPrefix(script, "///")` → `strings.HasPrefix(script, "///$tab")`
- Line 22: change the error message to `"expected script to start with ///$tab, got: ..."`
- Line 65: change `strings.HasPrefix(script, "///")` → `strings.HasPrefix(script, "///$tab")`
- Line 66: change the error message to `"expected script to start with ///$tab, got: ..."`

- [ ] **Step 2: Run tests**

```bash
go test ./internal/extractor/... -run "TestExtractScript_ValidFile|TestExtractScript_NoEndMarker" -v
```

Expected: both PASS (the fixture starts with `///$tab Main`).

- [ ] **Step 3: Commit**

```bash
git add internal/extractor/qvw_test.go
git commit -m "test: tighten prefix assertions to ///$tab in qvw tests"
git push
```

---

### Task 7: Update Integration Tests

**Files:**
- Modify: `internal/extractor/qlikview_integration_test.go`

- [ ] **Step 1: Update `TestQlikview_WalkerFindsAllFiles`**

Change line 32 from `len(paths) != 1` to `len(paths) != 2` and update the error message from `"expected 1 QVW file"` to `"expected 2 files (1 QVW + 1 QVF)"`.

- [ ] **Step 2: Update `TestQlikview_AllFilesExtractWithoutError`**

Replace the call to `extractor.ExtractScript(p)` with a dispatch:

```go
var err error
switch filepath.Ext(p) {
case ".qvf":
    _, err = extractor.ExtractScriptFromQVF(p)
default:
    _, err = extractor.ExtractScript(p)
}
```

Add `"path/filepath"` to imports if not already present.

- [ ] **Step 3: Update `TestQlikview_AllScriptsStartWithTripleSlash`**

Replace:
```go
script, err := extractor.ExtractScript(p)
```
with:
```go
var script string
var err error
switch filepath.Ext(p) {
case ".qvf":
    script, err = extractor.ExtractScriptFromQVF(p)
default:
    script, err = extractor.ExtractScript(p)
}
```

Change the assertion from `strings.HasPrefix(script, "///")` to `strings.HasPrefix(script, "///$tab")` and update the error message to `"expected script to start with ///$tab, got: ..."`.

Note: `filepath` is already imported in this file.

- [ ] **Step 4: Update `TestQlikview_ExtractSucceeds_ExitCode0`**

Change line 111 from `strings.Contains(out, "Extracted 1 scripts")` to `strings.Contains(out, "Extracted 2 scripts")`.

- [ ] **Step 5: Run integration tests**

```bash
go test ./internal/extractor/... -v -run "TestQlikview"
```

Expected: all 5 integration tests PASS.

- [ ] **Step 6: Run full test suite**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 7: Commit**

```bash
git add internal/extractor/qlikview_integration_test.go
git commit -m "test: update integration tests for QVF support"
git push
```
