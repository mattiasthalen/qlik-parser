# QlikView Script Extractor — Phase 02: Core Extractor Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `internal/extractor/` — the three pure-Go packages (walker, qvw, exporter) with >80% test coverage and no UI dependencies.

**Architecture:** Three focused files, each with a single responsibility. All operate on plain Go types — no cobra, no bubbletea. `walker.go` returns `[]string`. `qvw.go` takes a file path and returns a script string. `exporter.go` resolves paths and writes files. Tests use `testdata/` fixtures and temp dirs.

**Tech Stack:** Go 1.24 stdlib (`compress/zlib`, `io/fs`, `path/filepath`, `strings`, `os`, `bytes`, `regexp`), no external deps.

**Spec:** `docs/superpowers/specs/2026-03-11-qlik-script-extractor-design.md`

**Prerequisites:** Phase 01 complete (`go.mod` exists, module path known).

**Deferred to Phase 04:** `--source is a file → exit 1` and `--out` top-level pre-flight `MkdirAll` failure are command-layer concerns tested in `cmd/export_test.go` (Phase 04), not in `internal/extractor/`.

**Parallelism note:** Tasks 1–3 (walker) are self-contained. Tasks 4–6 (qvw) are self-contained. Tasks 7–9 (exporter) depend on qvw types being defined (Task 4). Tasks 1–3 and 4–6 can be executed in parallel worktrees.

---

## Chunk 1: Walker

### Task 1: Create testdata structure and walker test

**Files:**
- Create: `internal/extractor/testdata/simple/a.qvw` (binary fixture — see step 1)
- Create: `internal/extractor/testdata/sub/b.qvw` (binary fixture — see step 1)
- Create: `internal/extractor/walker_test.go`

- [ ] **Step 1: Create testdata directories and placeholder fixtures**

The fixtures don't need to be real QVW files for walker tests — they just need to exist with the `.qvw` extension.

```bash
mkdir -p internal/extractor/testdata/simple
mkdir -p internal/extractor/testdata/sub
# Create minimal placeholder files (not valid QVW, but walker only checks extension)
printf '\x00' > internal/extractor/testdata/simple/a.qvw
printf '\x00' > internal/extractor/testdata/sub/b.qvw
```

- [ ] **Step 2: Write walker tests**

```go
package extractor_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/your-org/qlik-script-extractor/internal/extractor"
)

func TestWalkFindsQVWFiles(t *testing.T) {
	root := t.TempDir()
	// Create nested .qvw files
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
		filepath.Join(root, "a", "ignore.txt"), // must not appear
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
		os.WriteFile(filepath.Join(root, name), []byte{0x00}, 0644)
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
	t.Cleanup(func() { os.Chmod(denied, 0755) })

	_, warns := extractor.Walk(root)
	if len(warns) == 0 {
		t.Error("expected at least one warn for unreadable subdir, got none")
	}
}

func TestWalkDoesNotFollowSymlinks(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	// Put a .qvw in the symlink target
	os.WriteFile(filepath.Join(target, "linked.qvw"), []byte{0x00}, 0644)
	// Symlink from root into target
	os.Symlink(target, filepath.Join(root, "link"))

	got, _ := extractor.Walk(root)
	for _, f := range got {
		if filepath.Base(f) == "linked.qvw" {
			t.Error("Walk followed a symlink — expected it to be skipped")
		}
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/extractor/... -run TestWalk -v`
Expected: FAIL — `extractor` package does not exist.

---

### Task 2: Implement walker.go

**Files:**
- Create: `internal/extractor/walker.go`

- [ ] **Step 1: Write walker.go**

```go
package extractor

import (
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/rs/zerolog/log"
)

// Walk recursively collects all *.qvw file paths under root.
// Returns sorted paths and a slice of warning messages for unreadable directories.
// Symlinks are not followed.
func Walk(root string) (paths []string, warns []string) {
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			warns = append(warns, path+": "+err.Error())
			log.Warn().Str("path", path).Err(err).Msg("skipping unreadable path")
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip symlinks to directories (WalkDir does not follow them by default,
		// but symlinks to files would still be walked — skip those too).
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if !d.IsDir() && filepath.Ext(path) == ".qvw" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		warns = append(warns, root+": "+err.Error())
	}
	sort.Strings(paths)
	return paths, warns
}
```

- [ ] **Step 2: Run walker tests**

Run: `go test ./internal/extractor/... -run TestWalk -v`
Expected: All PASS

- [ ] **Step 3: Commit**

```bash
git add internal/extractor/walker.go internal/extractor/walker_test.go \
        internal/extractor/testdata/
git commit -m "feat: implement recursive QVW file walker"
```

---

## Chunk 2: QVW Decompressor and Script Extractor

### Task 3: Create real QVW test fixtures

**Files:**
- Create: `internal/extractor/testdata/fixtures/` (directory with binary fixtures)
- Create: `internal/extractor/testcreate/gen_fixtures_test.go` (helper to create synthetic fixtures)

The QVW format: first 23 bytes are a header, then zlib-compressed data. We create fixtures programmatically in a test helper so we don't need to ship binary blobs.

- [ ] **Step 1: Create fixture generator helper**

```bash
mkdir -p internal/extractor/testdata/fixtures
```

Create `internal/extractor/testdata/gen/main.go`:

```go
//go:build ignore

package main

// Run: go run internal/extractor/testdata/gen/main.go
// Generates binary .qvw fixture files for tests.

import (
	"bytes"
	"compress/zlib"
	"os"
)

func makeQVW(payload []byte) []byte {
	header := make([]byte, 23) // arbitrary placeholder header
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(payload)
	w.Close()
	return append(header, buf.Bytes()...)
}

func write(path string, data []byte) {
	if err := os.WriteFile(path, data, 0644); err != nil {
		panic(err)
	}
}

func main() {
	dir := "internal/extractor/testdata/fixtures"

	// valid.qvw: has a script between /// and end marker
	script := []byte("///\nLOAD * FROM table.csv;\n")
	payload := append(script, []byte("\r\n\x00\x00\x00")...)
	write(dir+"/valid.qvw", makeQVW(payload))

	// no_script.qvw: no /// marker
	write(dir+"/no_script.qvw", makeQVW([]byte("some binary data without triple slash")))

	// no_end_marker.qvw: /// found but no end marker — script is full 100k region
	longScript := make([]byte, 0, 200)
	longScript = append(longScript, []byte("///\nLOAD * FROM big_table;")...)
	write(dir+"/no_end_marker.qvw", makeQVW(longScript))

	// invalid_zlib.qvw: valid header size but garbage compressed data
	garbage := make([]byte, 50)
	for i := range garbage {
		garbage[i] = 0xFF
	}
	header := make([]byte, 23)
	write(dir+"/invalid_zlib.qvw", append(header, garbage...))

	// too_short.qvw: fewer than 23 bytes
	write(dir+"/too_short.qvw", []byte("short"))

	// invalid_utf8.qvw: script contains bytes that are invalid UTF-8
	utf8Payload := []byte("///\nLOAD \xFF\xFE * FROM table;\n\r\n\x00\x00")
	write(dir+"/invalid_utf8.qvw", makeQVW(utf8Payload))
}
```

- [ ] **Step 2: Run the generator**

```bash
go run internal/extractor/testdata/gen/main.go
```

Expected: Six `.qvw` files created in `internal/extractor/testdata/fixtures/`.

- [ ] **Step 3: Verify fixture files**

Run: `ls -la internal/extractor/testdata/fixtures/`
Expected: `valid.qvw`, `no_script.qvw`, `no_end_marker.qvw`, `invalid_zlib.qvw`, `too_short.qvw`, `invalid_utf8.qvw`

- [ ] **Step 4: Commit generator and fixtures**

```bash
git add internal/extractor/testdata/
git commit -m "test: add QVW fixture generator and binary fixtures"
```

---

### Task 4: Write qvw.go tests

**Files:**
- Create: `internal/extractor/qvw_test.go`

- [ ] **Step 1: Write tests for ExtractScript**

```go
package extractor_test

import (
	"strings"
	"testing"

	"github.com/your-org/qlik-script-extractor/internal/extractor"
)

const fixturesDir = "testdata/fixtures"

func TestExtractScript_ValidFile(t *testing.T) {
	script, err := extractor.ExtractScript(fixturesDir + "/valid.qvw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(script, "///") {
		t.Errorf("expected script to start with ///, got: %q", script[:min(20, len(script))])
	}
	if !strings.Contains(script, "LOAD * FROM table.csv") {
		t.Errorf("expected script content, got: %q", script)
	}
}

func TestExtractScript_TooShort(t *testing.T) {
	_, err := extractor.ExtractScript(fixturesDir + "/too_short.qvw")
	if err == nil {
		t.Fatal("expected error for too-short file, got nil")
	}
	if !strings.Contains(err.Error(), "file too short") {
		t.Errorf("expected 'file too short' error, got: %v", err)
	}
}

func TestExtractScript_InvalidZlib(t *testing.T) {
	_, err := extractor.ExtractScript(fixturesDir + "/invalid_zlib.qvw")
	if err == nil {
		t.Fatal("expected error for invalid zlib, got nil")
	}
}

func TestExtractScript_NoScriptMarker(t *testing.T) {
	_, err := extractor.ExtractScript(fixturesDir + "/no_script.qvw")
	if err == nil {
		t.Fatal("expected error for missing /// marker, got nil")
	}
	var noScript *extractor.NoScriptError
	if !extractor.IsNoScript(err, &noScript) {
		t.Errorf("expected NoScriptError, got: %T %v", err, err)
	}
}

func TestExtractScript_NoEndMarker(t *testing.T) {
	// Should succeed — full region used when no end marker
	script, err := extractor.ExtractScript(fixturesDir + "/no_end_marker.qvw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(script, "///") {
		t.Errorf("expected script to start with ///, got: %q", script)
	}
}

func TestExtractScript_TruncatesAt100k(t *testing.T) {
	// Build a synthetic .qvw in memory: 23-byte header + zlib(/// + 200k bytes)
	// Script should be truncated to 100,000 bytes.
	import_bytes := func() []byte {
		var buf bytes.Buffer
		w := zlib.NewWriter(&buf)
		payload := make([]byte, 200_100)
		copy(payload, []byte("///"))
		for i := 3; i < len(payload); i++ {
			payload[i] = 'X'
		}
		w.Write(payload)
		w.Close()
		header := make([]byte, 23)
		return append(header, buf.Bytes()...)
	}

	// Write to temp file
	f, err := os.CreateTemp("", "truncate_test_*.qvw")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Write(import_bytes())
	f.Close()

	script, err := extractor.ExtractScript(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(script) > 100_000 {
		t.Errorf("expected script truncated to 100,000 bytes, got %d", len(script))
	}
}

// Note: the test above uses compress/zlib and bytes inline.
// Add these imports to qvw_test.go:
//   "bytes"
//   "compress/zlib"
//   "os"

func TestExtractScript_InvalidUTF8(t *testing.T) {
	// Invalid UTF-8 bytes should be replaced with replacement character
	script, err := extractor.ExtractScript(fixturesDir + "/invalid_utf8.qvw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should contain replacement character \uFFFD
	if !strings.Contains(script, "\uFFFD") {
		t.Errorf("expected replacement character in output for invalid UTF-8, got: %q", script)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/extractor/... -run TestExtractScript -v`
Expected: FAIL — `ExtractScript` not defined.

---

### Task 5: Implement qvw.go

**Files:**
- Create: `internal/extractor/qvw.go`

- [ ] **Step 1: Write qvw.go**

```go
package extractor

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const headerSize = 23
const maxScriptRegion = 100_000

// NoScriptError is returned when no /// marker is found in the decompressed data.
type NoScriptError struct {
	Path string
}

func (e *NoScriptError) Error() string {
	return fmt.Sprintf("%s: no script found", e.Path)
}

// IsNoScript reports whether err is a *NoScriptError and sets target if so.
func IsNoScript(err error, target **NoScriptError) bool {
	return errors.As(err, target)
}

// ExtractScript reads a .qvw file, decompresses its body, and returns the
// embedded load script as a UTF-8 string.
//
// Errors:
//   - "file too short" if file is < 23 bytes
//   - zlib error on decompression failure
//   - *NoScriptError if no /// marker is found
func ExtractScript(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	if len(data) < headerSize {
		return "", fmt.Errorf("%s: file too short", path)
	}

	compressed := data[headerSize:]
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return "", fmt.Errorf("%s: zlib: %w", path, err)
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("%s: zlib: %w", path, err)
	}

	return extractFromBytes(path, decompressed)
}

// extractFromBytes extracts the script from raw decompressed bytes.
func extractFromBytes(path string, data []byte) (string, error) {
	marker := []byte("///")
	scriptStart := bytes.Index(data, marker)
	if scriptStart < 0 {
		return "", &NoScriptError{Path: path}
	}

	end := scriptStart + maxScriptRegion
	if end > len(data) {
		end = len(data)
	}
	region := data[scriptStart:end]

	scriptBytes := trimAtEndMarker(region)
	return strings.ToValidUTF8(string(scriptBytes), "\uFFFD"), nil
}

// trimAtEndMarker finds the end of the script region:
// a newline (\r\n or \n) followed by two or more \x00 bytes.
// Returns region up to (not including) the trailing newline.
func trimAtEndMarker(region []byte) []byte {
	// Search for \r\n\x00\x00 or \n\x00\x00
	for i := 0; i < len(region)-2; i++ {
		if region[i] == '\n' {
			// Check if preceded by \r
			nlStart := i
			if i > 0 && region[i-1] == '\r' {
				nlStart = i - 1
			}
			// Count consecutive \x00 after \n
			j := i + 1
			for j < len(region) && region[j] == 0x00 {
				j++
			}
			if j-i-1 >= 2 { // at least 2 null bytes after \n
				return region[:nlStart]
			}
		}
	}
	return region
}
```

- [ ] **Step 2: Run qvw tests**

Run: `go test ./internal/extractor/... -run TestExtractScript -v`
Expected: All PASS

- [ ] **Step 3: Check coverage for qvw.go**

Run: `go test ./internal/extractor/... -coverprofile=coverage.out && go tool cover -func=coverage.out | grep qvw`
Expected: Coverage for `qvw.go` is >80%.

- [ ] **Step 4: Commit**

```bash
git add internal/extractor/qvw.go internal/extractor/qvw_test.go
git commit -m "feat: implement QVW decompressor and script extractor"
```

---

## Chunk 3: Exporter

### Task 6: Write exporter tests

**Files:**
- Create: `internal/extractor/exporter_test.go`

- [ ] **Step 1: Write exporter tests**

```go
package extractor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/qlik-script-extractor/internal/extractor"
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
	// When --out == --source, acts like alongside mode
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/extractor/... -run TestResolveOutput -run TestWriteScript -v`
Expected: FAIL — `ResolveOutputPath` and `WriteScript` not defined.

---

### Task 7: Implement exporter.go

**Files:**
- Create: `internal/extractor/exporter.go`

- [ ] **Step 1: Write exporter.go**

```go
package extractor

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolveOutputPath computes the destination .qvs path for a given .qvw.
//
//   - qvwPath:   absolute path to the source .qvw file
//   - sourceDir: the --source directory (used to compute relative path)
//   - outDir:    the --out directory; empty string or equal to sourceDir → alongside mode
func ResolveOutputPath(qvwPath, sourceDir, outDir string) string {
	base := strings.TrimSuffix(filepath.Base(qvwPath), ".qvw") + ".qvs"

	if outDir == "" || outDir == sourceDir {
		// alongside mode
		return filepath.Join(filepath.Dir(qvwPath), base)
	}

	// mirror mode: preserve directory structure relative to sourceDir
	rel, err := filepath.Rel(sourceDir, filepath.Dir(qvwPath))
	if err != nil {
		// fallback: place directly in outDir
		return filepath.Join(outDir, base)
	}
	return filepath.Join(outDir, rel, base)
}

// WriteScript writes script to outPath. In dry-run mode it is a no-op.
// Intermediate directories are created automatically.
func WriteScript(outPath, script string, dryRun bool) error {
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(outPath, []byte(script), 0644)
}
```

- [ ] **Step 2: Run exporter tests**

Run: `go test ./internal/extractor/... -run TestResolveOutput -run TestWriteScript -v`
Expected: All PASS

- [ ] **Step 3: Run all extractor tests**

Run: `go test ./internal/extractor/... -v`
Expected: All PASS

- [ ] **Step 4: Check overall coverage**

Run: `go test ./internal/extractor/... -coverprofile=coverage.out && go tool cover -func=coverage.out`
Expected: Total coverage >80%.

- [ ] **Step 5: Commit**

```bash
git add internal/extractor/exporter.go internal/extractor/exporter_test.go
git commit -m "feat: implement output path resolver and script writer"
```

---

### Task 8: Final extractor package validation

- [ ] **Step 1: Run full test suite**

Run: `make test`
Expected: All tests in all packages pass.

- [ ] **Step 2: Run linter**

Run: `make lint`
Expected: No lint errors. If golangci-lint reports issues, fix them and re-run before committing.

- [ ] **Step 3: Commit any lint fixes**

```bash
git add -p
git commit -m "fix: resolve linter warnings in extractor package"
```

(Only create this commit if there were lint fixes to make.)
