# Rebrand to qlik-parser Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename the tool from `qlik-script-extractor` to `qlik-parser`, rename the `export` command to `extract`, and add a `--script` flag with "no artifact selected" validation.

**Architecture:** Pure rename + CLI restructure. The Go module path changes throughout, `cmd/export.go` becomes `cmd/extract.go` with the new flag and validation guard, and tooling files (Makefile, .gitignore, .goreleaser.yaml) are updated. No changes to `internal/extractor` or `internal/ui`.

**Tech Stack:** Go 1.25, Cobra (CLI framework), standard `go mod` tooling.

---

## Chunk 1: Module rename and tooling

### Task 1: Update go.mod, run go mod tidy, update main.go

**Files:**
- Modify: `go.mod`
- Modify: `go.sum` (via `go mod tidy`)
- Modify: `main.go`

- [ ] **Step 1: Update go.mod module path**

In `go.mod`, change line 1:
```
module github.com/mattiasthalen/qlik-script-extractor
```
to:
```
module github.com/mattiasthalen/qlik-parser
```

- [ ] **Step 2: Run go mod tidy**

```bash
go mod tidy
```
Expected: no errors, `go.sum` updated (or unchanged if no deps changed).

- [ ] **Step 3: Update main.go import**

In `main.go` line 7, change:
```go
"github.com/mattiasthalen/qlik-script-extractor/cmd"
```
to:
```go
"github.com/mattiasthalen/qlik-parser/cmd"
```

- [ ] **Step 4: Verify it still compiles**

```bash
go build ./...
```
Expected: build errors for all files still using old import path (we'll fix those next). If only `main.go` errors remain about undefined `cmd`, that's wrong — `main.go` should compile after this step since we just updated it. Confirm no errors from `main.go` specifically.

---

### Task 2: Update all remaining Go import paths

**Files:**
- Modify: `cmd/root.go`
- Modify: `cmd/version.go`
- Modify: `cmd/root_test.go`
- Modify: `cmd/version_test.go`
- Modify: `internal/extractor/exporter.go` (no import to change, but verify)
- Modify: `internal/extractor/exporter_test.go`
- Modify: `internal/extractor/qvw.go` (no import to change, but verify)
- Modify: `internal/extractor/qvw_test.go`
- Modify: `internal/extractor/walker.go` (no import to change, but verify)
- Modify: `internal/extractor/walker_test.go`
- Modify: `internal/extractor/qlikview_integration_test.go`
- Modify: `internal/extractor/testdata/gen/golden_gen.go`
- Modify: `internal/ui/output.go` (no import to change, but verify)
- Modify: `internal/ui/output_test.go`
- Modify: `internal/ui/tty.go` (no import to change, but verify)
- Modify: `internal/ui/tty_test.go`

- [ ] **Step 1: Find all files with old import path**

```bash
grep -r "qlik-script-extractor" --include="*.go" -l
```
Expected output (verify all these are present — `cmd/root.go` and `cmd/version.go` will also appear because they contain the literal string as a value, not just as an import path; those are handled in Task 4):
```
cmd/root.go
cmd/version.go
cmd/version_test.go
cmd/root_test.go
internal/extractor/exporter_test.go
internal/extractor/walker_test.go
internal/extractor/qvw_test.go
internal/extractor/qlikview_integration_test.go
internal/extractor/testdata/gen/golden_gen.go
internal/ui/output_test.go
internal/ui/tty_test.go
```
(main.go was already updated in Task 1; `cmd/export.go` and `cmd/export_test.go` will also appear but are handled in Task 5)

- [ ] **Step 2: Replace old import path in all files**

For each file listed above, change every occurrence of:
```
github.com/mattiasthalen/qlik-script-extractor
```
to:
```
github.com/mattiasthalen/qlik-parser
```

Note: `cmd/export.go` and `cmd/export_test.go` will be handled in Task 5 (they get renamed and rewritten). `cmd/root.go` and `cmd/version.go` contain the old string as values (not import paths) — those are updated in Task 4, not here.

- [ ] **Step 3: Verify all Go files compile**

```bash
go build ./...
```
Expected: build errors from `cmd/export.go` (import path not yet updated) and possibly `cmd/root.go` (references `newExportCmd` which still exists). All other packages should compile cleanly. This is expected at this stage — Tasks 4 and 5 will resolve the remaining errors.

---

### Task 3: Update tooling files

**Files:**
- Modify: `Makefile`
- Modify: `.gitignore`
- Modify: `.goreleaser.yaml`

- [ ] **Step 1: Update Makefile**

In `Makefile` line 3, change:
```makefile
BINARY := qlik-script-extractor
```
to:
```makefile
BINARY := qlik-parser
```

- [ ] **Step 2: Update .gitignore**

In `.gitignore` line 5, change:
```
qlik-script-extractor
```
to:
```
qlik-parser
```

- [ ] **Step 3: Update .goreleaser.yaml**

Change line 8:
```yaml
  - binary: qlik-script-extractor
```
to:
```yaml
  - binary: qlik-parser
```

Change line 10:
```yaml
      - -X github.com/mattiasthalen/qlik-script-extractor/cmd.Version={{ .Version }}
```
to:
```yaml
      - -X github.com/mattiasthalen/qlik-parser/cmd.Version={{ .Version }}
```

- [ ] **Step 4: Verify make build produces correctly-named binary**

```bash
make build
ls qlik-parser
```
Expected: `qlik-parser` binary exists.

```bash
make clean
ls qlik-parser 2>/dev/null || echo "cleaned"
```
Expected: `cleaned`

---

## Chunk 2: Root and version commands

### Task 4: Update root.go and version.go

**Files:**
- Modify: `cmd/root.go`
- Modify: `cmd/version.go`

- [ ] **Step 1: Update root.go**

In `cmd/root.go`, make these changes:

Change `Use` field:
```go
Use:   "qlik-script-extractor",
```
to:
```go
Use:   "qlik-parser",
```

Change `Short`:
```go
Short: "Extract QlikView load scripts from .qvw files",
```
to:
```go
Short: "Parse and extract artifacts from QlikView .qvw files",
```

Change `Long`:
```go
Long: `qlik-script-extractor recursively scans a directory for QVW files
and extracts the embedded load scripts to .qvs text files.`,
```
to:
```go
Long: `qlik-parser recursively scans a directory for QVW files
and extracts embedded artifacts (load scripts, and more to come).`,
```

Change `AddCommand` call:
```go
root.AddCommand(newExportCmd())
```
to:
```go
root.AddCommand(newExtractCmd())
```

- [ ] **Step 2: Update version.go**

In `cmd/version.go` line 17, change:
```go
_, _ = fmt.Fprintf(cmd.OutOrStdout(), "qlik-script-extractor %s\n", Version)
```
to:
```go
_, _ = fmt.Fprintf(cmd.OutOrStdout(), "qlik-parser %s\n", Version)
```

- [ ] **Step 3: Update root_test.go and version_test.go assertions**

In `cmd/root_test.go` line 19, change:
```go
if !bytes.Contains(buf.Bytes(), []byte("qlik-script-extractor")) {
```
to:
```go
if !bytes.Contains(buf.Bytes(), []byte("qlik-parser")) {
```

In `cmd/version_test.go` line 20, change:
```go
if !bytes.Contains([]byte(out), []byte("qlik-script-extractor")) {
```
to:
```go
if !bytes.Contains([]byte(out), []byte("qlik-parser")) {
```

- [ ] **Step 4: Run tests to verify**

```bash
go test ./cmd/... -run "TestRootHelp|TestVersionCmd" -v
```
Expected: both tests PASS.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum main.go cmd/root.go cmd/version.go cmd/root_test.go cmd/version_test.go Makefile .gitignore .goreleaser.yaml
git add internal/extractor/ internal/ui/
git commit -m "chore: rename module and binary to qlik-parser"
git push
```

---

## Chunk 3: extract command (rename + --script flag)

### Task 5: Write the failing test for --script=false validation

**Files:**
- Create: `cmd/extract_test.go` (copy of `cmd/export_test.go` with updates)

- [ ] **Step 1: Copy export_test.go to extract_test.go**

```bash
cp cmd/export_test.go cmd/extract_test.go
```

- [ ] **Step 2: Update extract_test.go — rename functions and update "export" → "extract"**

Make the following changes in `cmd/extract_test.go`:

1. Change package import path (if still old — should already be updated from Task 2):
   ```go
   "github.com/mattiasthalen/qlik-parser/cmd"
   ```

2. Rename all test functions: `TestExportCmd_*` → `TestExtractCmd_*`

3. In every `root.SetArgs([]string{...})` call, change `"export"` to `"extract"`:
   - `{"export", "--help"}` → `{"extract", "--help"}`
   - `{"export", "--source", ...}` → `{"extract", "--source", ...}`
   - etc. (8 occurrences total)

4. Add the new failing test at the end of the file:

```go
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
```

- [ ] **Step 3: Delete export_test.go**

Both files are in `package cmd_test` and both declare `var update = flag.Bool(...)` — having both present causes a compile error. Delete the old file before running tests.

```bash
rm cmd/export_test.go
```

- [ ] **Step 4: Run the new test to verify it fails**

```bash
go test ./cmd/... -run "TestExtractCmd_NoArtifactSelected" -v
```
Expected: FAIL — the `extract` command doesn't exist yet, so cobra will return "unknown command". This confirms the test is exercising real behaviour.

---

### Task 6: Create extract.go (rename + --script flag + validation)

**Files:**
- Create: `cmd/extract.go` (based on `cmd/export.go`)
- Delete: `cmd/export.go`

- [ ] **Step 1: Copy export.go to extract.go**

```bash
cp cmd/export.go cmd/extract.go
```

- [ ] **Step 2: Update extract.go**

Make these changes in `cmd/extract.go`:

1. Change the import path (if not already updated):
   ```go
   "github.com/mattiasthalen/qlik-parser/internal/extractor"
   "github.com/mattiasthalen/qlik-parser/internal/ui"
   ```

2. Rename the constructor function:
   ```go
   func newExportCmd() *cobra.Command {
   ```
   to:
   ```go
   func newExtractCmd() *cobra.Command {
   ```

3. Add `script` variable alongside existing vars at the top of the function:
   ```go
   var sourceDir string
   var outDir string
   var dryRun bool
   var script bool
   ```

4. Rename the inner loop variable `script` (the extracted script content) to `scriptContent` — it conflicts with the new `var script bool`. This variable appears in three places inside the `for` loop:

   Line ~72:
   ```go
   script, extractErr := extractor.ExtractScript(qvwPath)
   ```
   → rename to:
   ```go
   scriptContent, extractErr := extractor.ExtractScript(qvwPath)
   ```

   Line ~109:
   ```go
   writeErr := extractor.WriteScript(outPath, script, dryRun)
   ```
   → rename to:
   ```go
   writeErr := extractor.WriteScript(outPath, scriptContent, dryRun)
   ```

   Line ~126:
   ```go
   CharCount: len(script),
   ```
   → rename to:
   ```go
   CharCount: len(scriptContent),
   ```

5. Change `Use` field:
   ```go
   Use:   "export",
   ```
   to:
   ```go
   Use:   "extract",
   ```

5. Change `Short`:
   ```go
   Short: "Extract load scripts from .qvw files",
   ```
   to:
   ```go
   Short: "Extract artifacts from .qvw files",
   ```

6. Change `Long` (note: no leading spaces on the second line in the actual file):
   ```go
   Long: `Recursively scans --source for .qvw files and extracts the embedded
load scripts to .qvs text files alongside or under --out.`,
   ```
   to:
   ```go
   Long: `Recursively scans --source for .qvw files and extracts embedded
   artifacts to text files alongside or under --out.`,
   ```

7. Add the validation guard as the **first statement** inside `RunE`, before the `sourceDir` defaulting block:
   ```go
   RunE: func(cmd *cobra.Command, args []string) error {
       if !script { // expand to: if !script && !variables && !charts as flags are added
           _, _ = fmt.Fprintf(cmd.ErrOrStderr(), "error: no artifact type selected\n")
           return ExitError(1)
       }

       if sourceDir == "" {
       // ... rest of existing code unchanged
   ```

8. Add the `--script` flag registration at the end of the flag definitions:
   ```go
   cmd.Flags().BoolVar(&script, "script", true, "Extract load scripts")
   ```

- [ ] **Step 3: Delete export.go**

```bash
rm cmd/export.go
```

- [ ] **Step 4: Run the new test to verify it passes**

```bash
go test ./cmd/... -run "TestExtractCmd_NoArtifactSelected" -v
```
Expected: PASS.

- [ ] **Step 5: Run all tests**

```bash
go test ./... -v -count=1
```
Expected: all tests PASS. No `TestExportCmd_*` tests should appear (those were renamed).

- [ ] **Step 6: Commit**

```bash
git add cmd/extract.go cmd/extract_test.go
git rm cmd/export.go cmd/export_test.go
git commit -m "feat: rename export→extract command, add --script flag (#4)"
git push
```

---

## Post-Implementation

### Task 7: Manual GitHub repo rename (user action)

- [ ] **Step 1: Rename repo on GitHub**

  Go to: GitHub → repository **Settings** → **General** → **Repository name** → type `qlik-parser` → click **Rename**.

- [ ] **Step 2: Update local remote URL**

```bash
git remote set-url origin https://github.com/mattiasthalen/qlik-parser
git remote -v
```
Expected: both fetch and push show `https://github.com/mattiasthalen/qlik-parser`.

- [ ] **Step 3: Verify push still works**

```bash
git push
```
Expected: success (GitHub redirects automatically after rename, but updating the URL is cleaner).
