# Artifact Extraction Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend `qlik-parser extract` to extract measures, dimensions, and variables from `.qvf` files alongside load scripts, with per-source-file output folders and updated flag semantics.

**Architecture:** A single-pass `ParseQVF` function replaces the script-only extractor; `ResolveOutputPath` becomes `ResolveOutputDir` returning a folder; `WriteScript` becomes `WriteArtifacts` accepting a slice of named byte payloads. The `Result` struct and `Printer` are updated to reflect folder + file-list output instead of path + char count. The `extract` command's flag logic switches from a `--script` bool defaulting `true` to four flags all defaulting `false` with "no flags = all" semantics.

**Tech Stack:** Go 1.25, `encoding/json`, `compress/zlib`, `github.com/spf13/cobra`, `github.com/charmbracelet/lipgloss`

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `internal/extractor/qvf.go` | Modify | Add `ParseQVF`, `QVFData`, `Measure`, `Dimension`, `Variable` types; keep `ExtractScriptFromQVF` as a thin wrapper |
| `internal/extractor/qvf_test.go` | Modify | Add `ParseQVF` tests (script, measures, dimensions, variables, empty lists) |
| `internal/extractor/exporter.go` | Modify | Replace `ResolveOutputPath`→`ResolveOutputDir`, `WriteScript`→`WriteArtifacts`; add `Artifact` type |
| `internal/extractor/exporter_test.go` | Modify | Rename `TestResolveOutputPath_*`→`TestResolveOutputDir_*`; replace `TestWriteScript_*`→`TestWriteArtifacts_*` |
| `internal/extractor/qlikview_integration_test.go` | Modify | Update summary assertion; add QVF artifact count/spot-check tests; add QVW JSON-absent test |
| `internal/ui/output.go` | Modify | Replace `QVSPath`/`CharCount` with `OutDir`/`Files` in `Result`; update `printOK` and `Summary` |
| `internal/ui/output_test.go` | Modify | Update `TestPrinter_SuccessLine`, `TestPrinter_Summary_*` to use new `Result` fields |
| `cmd/extract.go` | Modify | Replace `--script` bool (default `true`) with four flags (default `false`); switch to `ParseQVF`+`WriteArtifacts`+`ResolveOutputDir` |
| `cmd/extract_test.go` | Modify | Update `TestExtractCmd_Integration_ValidFixture` path; update `TestExtractCmd_DryRunNoFilesWritten`; `TestExtractCmd_NoArtifactSelected` needs no change |
| `internal/extractor/testdata/fixtures/measures.json.golden` | Create | Golden file for measure output spot-check |
| `internal/extractor/testdata/fixtures/dimensions.json.golden` | Create | Golden file for dimension output spot-check |
| `internal/extractor/testdata/fixtures/variables.json.golden` | Create | Golden file for variable output spot-check |

---

## Chunk 1: QVF Parser — `ParseQVF` and data types

### Task 1: Add `QVFData`, `Measure`, `Dimension`, `Variable` types and `ParseQVF` to `qvf.go`

**Files:**
- Modify: `internal/extractor/qvf.go`
- Modify: `internal/extractor/qvf_test.go`

**Context:** The current `ExtractScriptFromQVF` does a single-pass scan looking only for `qScript`. We need a new `ParseQVF` that in one pass also collects measures, dimensions, and variables. `ExtractScriptFromQVF` becomes a one-liner wrapper so existing tests keep passing.

The block detection rules (from the spec):
- `qScript` key present and non-empty → `QVFData.Script` (first match wins)
- `qInfo.qType == "measure"` → append to `QVFData.Measures`
- `qInfo.qType == "dimension"` → append to `QVFData.Dimensions`
- `qId == "user_variablelist"` → populate `QVFData.Variables` from `qEntryList`

For variables, the source JSON block looks like:
```json
{"qId":"user_variablelist","qEntryList":[{"qInfo":{"qId":"<uuid>","qType":"variable"},"qData":{"qName":"vMyVar","qComment":"","qValue":180}},...]}
```
Map `qData.qName` → `Variable.Name`, `qData.qComment` → `Variable.Comment`, `qData.qValue` (raw) → `Variable.Value`.

For measures, the source JSON block looks like:
```json
{"qInfo":{"qId":"EHphN","qType":"measure"},"qMeasure":{"qLabel":"Sessions","qDef":"Count(DISTINCT SessionID)","qTags":[],"qGrouping":0},"qMetaDef":{"title":"","description":"","tags":[]}}
```
Map: `qInfo.qId` → `Measure.ID`, `qMeasure.qLabel` → `Measure.Label`, `qMeasure.qDef` → `Measure.Def`, `qMeasure.qTags` → `Measure.Tags`, `qMetaDef.description` → `Measure.Description`.

For dimensions:
```json
{"qInfo":{"qId":"xdLfma","qType":"dimension"},"qDim":{"qFieldDefs":["AppName"],"qFieldLabels":[""],"qGrouping":0},"qDimInfos":[],"qMetaDef":{"title":"Application","description":"","tags":[]}}
```
Map: `qInfo.qId` → `Dimension.ID`, `qMetaDef.title` → `Dimension.Label`, `qDim.qFieldDefs` → `Dimension.Fields`, `qMetaDef.tags` → `Dimension.Tags`, `qMetaDef.description` → `Dimension.Description`.

- [ ] **Step 1: Write the failing tests for `ParseQVF`**

In `internal/extractor/qvf_test.go`, add after the existing tests.

The existing helper is `buildQVFFixture(t *testing.T, jsonPayload []byte) []byte` — it marshals JSON bytes into a fake QVF binary (64-byte junk prefix + one zlib stream). Add a `makeQVFFile` convenience wrapper that accepts `map[string]any`, marshals it, calls `buildQVFFixture`, and writes a temp file:

```go
// makeQVFFile marshals payload as JSON, builds a fake QVF binary via buildQVFFixture,
// and writes it to a temp file. Returns the file path.
func makeQVFFile(t *testing.T, payload map[string]any) string {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	data := buildQVFFixture(t, b)
	f, err := os.CreateTemp(t.TempDir(), "*.qvf")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	_, _ = f.Write(data)
	_ = f.Close()
	return f.Name()
}

func TestParseQVF_Script(t *testing.T) {
	path := makeQVFFile(t, map[string]any{
		"qScript": "LOAD * FROM t;",
	})
	got, err := extractor.ParseQVF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Script != "LOAD * FROM t;" {
		t.Errorf("Script = %q, want %q", got.Script, "LOAD * FROM t;")
	}
}

func TestParseQVF_Measures(t *testing.T) {
	path := makeQVFFile(t, map[string]any{
		"qInfo":    map[string]any{"qId": "abc", "qType": "measure"},
		"qMeasure": map[string]any{"qLabel": "Sales", "qDef": "Sum(Amount)", "qTags": []string{"tag1"}},
		"qMetaDef": map[string]any{"description": "Total sales"},
	})
	got, err := extractor.ParseQVF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Measures) != 1 {
		t.Fatalf("expected 1 measure, got %d", len(got.Measures))
	}
	m := got.Measures[0]
	if m.ID != "abc" || m.Label != "Sales" || m.Def != "Sum(Amount)" {
		t.Errorf("unexpected measure: %+v", m)
	}
	if len(m.Tags) != 1 || m.Tags[0] != "tag1" {
		t.Errorf("unexpected tags: %v", m.Tags)
	}
	if m.Description != "Total sales" {
		t.Errorf("unexpected description: %q", m.Description)
	}
}

func TestParseQVF_Dimensions(t *testing.T) {
	path := makeQVFFile(t, map[string]any{
		"qInfo":    map[string]any{"qId": "dim1", "qType": "dimension"},
		"qDim":     map[string]any{"qFieldDefs": []string{"Country"}, "qFieldLabels": []string{""}},
		"qMetaDef": map[string]any{"title": "Country", "description": "", "tags": []string{}},
	})
	got, err := extractor.ParseQVF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Dimensions) != 1 {
		t.Fatalf("expected 1 dimension, got %d", len(got.Dimensions))
	}
	d := got.Dimensions[0]
	if d.ID != "dim1" || d.Label != "Country" {
		t.Errorf("unexpected dimension: %+v", d)
	}
	if len(d.Fields) != 1 || d.Fields[0] != "Country" {
		t.Errorf("unexpected fields: %v", d.Fields)
	}
}

func TestParseQVF_Variables(t *testing.T) {
	path := makeQVFFile(t, map[string]any{
		"qId": "user_variablelist",
		"qEntryList": []map[string]any{
			{
				"qInfo": map[string]any{"qId": "uuid-1", "qType": "variable"},
				"qData": map[string]any{"qName": "vDaysBack", "qComment": "lookback", "qValue": 30},
			},
		},
	})
	got, err := extractor.ParseQVF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Variables) != 1 {
		t.Fatalf("expected 1 variable, got %d", len(got.Variables))
	}
	v := got.Variables[0]
	if v.ID != "uuid-1" || v.Name != "vDaysBack" || v.Comment != "lookback" {
		t.Errorf("unexpected variable: %+v", v)
	}
	// Value must be raw JSON — not re-encoded (number stays number)
	if string(v.Value) != "30" {
		t.Errorf("Value = %s, want 30", string(v.Value))
	}
}

func TestParseQVF_EmptyArtifactLists(t *testing.T) {
	// A file with a script block only — Measures/Dimensions/Variables should be non-nil empty slices
	path := makeQVFFile(t, map[string]any{"qScript": "LOAD 1;"})
	got, err := extractor.ParseQVF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Measures == nil || got.Dimensions == nil || got.Variables == nil {
		t.Error("expected non-nil empty slices for missing artifact types")
	}
}

func TestParseQVF_EmptyScript(t *testing.T) {
	// ParseQVF itself never returns NoScriptError — Script field is just empty.
	path := makeQVFFile(t, map[string]any{"other": "data"})
	got, err := extractor.ParseQVF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Script != "" {
		t.Errorf("expected empty Script, got %q", got.Script)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd /workspaces/qlik-parser/.worktrees/feat/artifact-extraction
go test ./internal/extractor/... -run "TestParseQVF" -v
```

Expected: FAIL — `ParseQVF: undefined`

- [ ] **Step 3: Implement `ParseQVF` and types in `internal/extractor/qvf.go`**

Add after the existing `qvfPayload` struct and `ExtractScriptFromQVF` function:

```go
// QVFData holds all artifacts extracted from a single .qvf file.
type QVFData struct {
	Script     string
	Measures   []Measure
	Dimensions []Dimension
	Variables  []Variable
}

// Measure represents a Qlik master measure.
type Measure struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Def         string   `json:"def"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
}

// Dimension represents a Qlik master dimension.
type Dimension struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Fields      []string `json:"fields"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
}

// Variable represents a Qlik variable.
type Variable struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Comment string          `json:"comment"`
	Value   json.RawMessage `json:"value"`
}

// ParseQVF reads a .qvf file and extracts all known artifact types in a single pass.
// It never returns NoScriptError; the Script field is simply empty if not found.
func ParseQVF(path string) (*QVFData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	result := &QVFData{
		Measures:   []Measure{},
		Dimensions: []Dimension{},
		Variables:  []Variable{},
	}

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
		trimmed := bytes.TrimRight(decompressed, "\x00")

		// Use a generic map to inspect top-level keys.
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &raw); err != nil {
			continue
		}

		// Script block
		if scriptRaw, ok := raw["qScript"]; ok && result.Script == "" {
			var s string
			if err := json.Unmarshal(scriptRaw, &s); err == nil && s != "" {
				result.Script = s
				continue
			}
		}

		// Variable list block
		if idRaw, ok := raw["qId"]; ok {
			var id string
			if err := json.Unmarshal(idRaw, &id); err == nil && id == "user_variablelist" {
				result.Variables = parseVariables(raw)
				continue
			}
		}

		// Measure or dimension block
		if infoRaw, ok := raw["qInfo"]; ok {
			var info struct {
				QID  string `json:"qId"`
				QType string `json:"qType"`
			}
			if err := json.Unmarshal(infoRaw, &info); err != nil {
				continue
			}
			switch info.QType {
			case "measure":
				if m, ok := parseMeasure(info.QID, raw); ok {
					result.Measures = append(result.Measures, m)
				}
			case "dimension":
				if d, ok := parseDimension(info.QID, raw); ok {
					result.Dimensions = append(result.Dimensions, d)
				}
			}
		}
	}

	return result, nil
}

func parseMeasure(id string, raw map[string]json.RawMessage) (Measure, bool) {
	var qMeasure struct {
		QLabel string   `json:"qLabel"`
		QDef   string   `json:"qDef"`
		QTags  []string `json:"qTags"`
	}
	if raw["qMeasure"] == nil {
		return Measure{}, false
	}
	if err := json.Unmarshal(raw["qMeasure"], &qMeasure); err != nil {
		return Measure{}, false
	}
	var meta struct {
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	}
	if raw["qMetaDef"] != nil {
		_ = json.Unmarshal(raw["qMetaDef"], &meta)
	}
	tags := qMeasure.QTags
	if tags == nil {
		tags = []string{}
	}
	return Measure{
		ID:          id,
		Label:       qMeasure.QLabel,
		Def:         qMeasure.QDef,
		Tags:        tags,
		Description: meta.Description,
	}, true
}

func parseDimension(id string, raw map[string]json.RawMessage) (Dimension, bool) {
	var qDim struct {
		QFieldDefs []string `json:"qFieldDefs"`
	}
	if raw["qDim"] == nil {
		return Dimension{}, false
	}
	if err := json.Unmarshal(raw["qDim"], &qDim); err != nil {
		return Dimension{}, false
	}
	var meta struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	}
	if raw["qMetaDef"] != nil {
		_ = json.Unmarshal(raw["qMetaDef"], &meta)
	}
	fields := qDim.QFieldDefs
	if fields == nil {
		fields = []string{}
	}
	tags := meta.Tags
	if tags == nil {
		tags = []string{}
	}
	return Dimension{
		ID:          id,
		Label:       meta.Title,
		Fields:      fields,
		Tags:        tags,
		Description: meta.Description,
	}, true
}

func parseVariables(raw map[string]json.RawMessage) []Variable {
	var list struct {
		QEntryList []struct {
			QInfo struct {
				QID string `json:"qId"`
			} `json:"qInfo"`
			QData struct {
				QName    string          `json:"qName"`
				QComment string          `json:"qComment"`
				QValue   json.RawMessage `json:"qValue"`
			} `json:"qData"`
		} `json:"qEntryList"`
	}
	if raw["qEntryList"] == nil {
		return []Variable{}
	}
	// Reconstruct the full JSON to unmarshal the entry list.
	full, err := json.Marshal(raw)
	if err != nil {
		return []Variable{}
	}
	if err := json.Unmarshal(full, &list); err != nil {
		return []Variable{}
	}
	vars := make([]Variable, 0, len(list.QEntryList))
	for _, e := range list.QEntryList {
		vars = append(vars, Variable{
			ID:      e.QInfo.QID,
			Name:    e.QData.QName,
			Comment: e.QData.QComment,
			Value:   e.QData.QValue,
		})
	}
	return vars
}
```

Also update `ExtractScriptFromQVF` to delegate to `ParseQVF`:

```go
// ExtractScriptFromQVF returns the embedded load script from a .qvf file.
// It delegates to ParseQVF and returns NoScriptError if no script is found.
func ExtractScriptFromQVF(path string) (string, error) {
	d, err := ParseQVF(path)
	if err != nil {
		return "", err
	}
	if d.Script == "" {
		return "", &NoScriptError{Path: path}
	}
	return d.Script, nil
}
```

Remove the old private `qvfPayload` struct and the old `ExtractScriptFromQVF` implementation (replaced above).

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd /workspaces/qlik-parser/.worktrees/feat/artifact-extraction
go test ./internal/extractor/... -run "TestParseQVF|TestExtractScriptFromQVF" -v
```

Expected: all PASS

- [ ] **Step 5: Run the full extractor test suite to check nothing regressed**

```bash
go test ./internal/extractor/... -v
```

Expected: all previously passing tests still PASS

- [ ] **Step 6: Commit**

```bash
git add internal/extractor/qvf.go internal/extractor/qvf_test.go
git commit -m "feat: add ParseQVF with measures, dimensions, and variables extraction"
git push
```

---

## Chunk 2: Exporter — `ResolveOutputDir` and `WriteArtifacts`

### Task 2: Replace `ResolveOutputPath`/`WriteScript` with `ResolveOutputDir`/`WriteArtifacts`

**Files:**
- Modify: `internal/extractor/exporter.go`
- Modify: `internal/extractor/exporter_test.go`

**Context:** `ResolveOutputPath` currently returns a file path (e.g. `/data/etl/sales.qvs`). The new `ResolveOutputDir` returns a folder path (e.g. `/data/etl/sales.qvw`) — i.e. the full input filename including extension becomes the folder name. `WriteArtifacts` writes a slice of `Artifact{Name, Content}` into the folder in a fail-fast manner.

- [ ] **Step 1: Replace all tests in `internal/extractor/exporter_test.go`**

The entire existing test file must be replaced. The old functions reference `ResolveOutputPath` and `WriteScript`, both of which are being deleted from `exporter.go`. Delete all eight old test functions:
- `TestResolveOutputPath_Alongside`
- `TestResolveOutputPath_Mirror`
- `TestResolveOutputPath_OutEqualSource`
- `TestResolveOutputPath_QVF_Alongside`
- `TestResolveOutputPath_QVF_Mirror`
- `TestWriteScript_CreatesFile`
- `TestWriteScript_DryRunDoesNotWrite`
- `TestWriteScript_CreatesIntermediateDirs`

Replace the entire file with the following (keeping the `package`, `import`, and adding the new tests):

```go
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
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test ./internal/extractor/... -run "TestResolveOutputDir|TestWriteArtifacts" -v
```

Expected: FAIL — `ResolveOutputDir: undefined`, `WriteArtifacts: undefined`, `Artifact: undefined`

- [ ] **Step 3: Implement `ResolveOutputDir`, `WriteArtifacts`, and `Artifact` in `internal/extractor/exporter.go`**

Replace the entire file content:

```go
package extractor

import (
	"os"
	"path/filepath"
)

// Artifact is a named file payload to be written into an output directory.
type Artifact struct {
	Name    string // filename, e.g. "script.qvs"
	Content []byte
}

// ResolveOutputDir computes the output folder path for a source file.
//
//   - inputPath: absolute path to the source file (.qvw or .qvf)
//   - sourceDir: the --source directory
//   - outDir:    the --out directory; empty string or equal to sourceDir → alongside mode
//
// The folder name is the full source filename including extension (e.g. "sales.qvw").
// No trailing slash is added (standard Go path convention).
func ResolveOutputDir(inputPath, sourceDir, outDir string) string {
	base := filepath.Base(inputPath) // e.g. "sales.qvw"

	if outDir == "" || outDir == sourceDir {
		return filepath.Join(filepath.Dir(inputPath), base)
	}

	rel, err := filepath.Rel(sourceDir, filepath.Dir(inputPath))
	if err != nil {
		return filepath.Join(outDir, base)
	}
	return filepath.Join(outDir, rel, base)
}

// WriteArtifacts writes each artifact into outDir. In dry-run mode it is a no-op.
// Intermediate directories are created automatically.
// Fail-fast: returns the first error encountered without attempting remaining writes.
func WriteArtifacts(outDir string, artifacts []Artifact, dryRun bool) error {
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	for _, a := range artifacts {
		if err := os.WriteFile(filepath.Join(outDir, a.Name), a.Content, 0644); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
go test ./internal/extractor/... -run "TestResolveOutputDir|TestWriteArtifacts" -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/extractor/exporter.go internal/extractor/exporter_test.go
git commit -m "feat: replace ResolveOutputPath/WriteScript with ResolveOutputDir/WriteArtifacts"
git push
```

---

## Chunk 3: UI — `Result` struct and `Printer` formatting

### Task 3: Update `Result` struct and `Printer` in `internal/ui/output.go`

**Files:**
- Modify: `internal/ui/output.go`
- Modify: `internal/ui/output_test.go`

**Context:** `Result` currently has `QVSPath string` and `CharCount int`. These are replaced by `OutDir string` (folder path, no trailing slash) and `Files []string` (artifact filenames in canonical order). `printOK` changes from `"src → qvs  (N chars)"` to `"src → outDir/  (file1, file2, ...)"`. `Summary` changes `"Extracted N scripts"` to `"Extracted N apps"`.

- [ ] **Step 1: Update the failing tests in `internal/ui/output_test.go`**

Replace the existing test functions that reference `QVSPath`, `CharCount`, or `"Extracted N scripts"`:

```go
func TestPrinter_SuccessLine(t *testing.T) {
	buf := &bytes.Buffer{}
	p := newTestPrinter(buf)
	p.FileResult(ui.Result{
		Status:  ui.StatusOK,
		SrcPath: "sales.qvw",
		OutDir:  "sales.qvw",
		Files:   []string{"script.qvs", "measures.json"},
	})
	out := buf.String()
	if !strings.Contains(out, "sales.qvw") {
		t.Errorf("expected src name in output, got: %q", out)
	}
	if !strings.Contains(out, "sales.qvw/") {
		t.Errorf("expected outDir with trailing slash in output, got: %q", out)
	}
	if !strings.Contains(out, "script.qvs") {
		t.Errorf("expected script.qvs in output, got: %q", out)
	}
	if !strings.Contains(out, "measures.json") {
		t.Errorf("expected measures.json in output, got: %q", out)
	}
}

func TestPrinter_Summary_Normal(t *testing.T) {
	buf := &bytes.Buffer{}
	p := newTestPrinter(buf)
	p.FileResult(ui.Result{Status: ui.StatusOK, SrcPath: "a.qvw", OutDir: "a.qvw", Files: []string{"script.qvs"}})
	p.FileResult(ui.Result{Status: ui.StatusOK, SrcPath: "b.qvw", OutDir: "b.qvw", Files: []string{"script.qvs"}})
	p.FileResult(ui.Result{Status: ui.StatusWarn, SrcPath: "c.qvw", Message: "no script found"})
	p.FileResult(ui.Result{Status: ui.StatusErr, SrcPath: "d.qvw", Message: "corrupt"})
	p.Summary()
	out := buf.String()
	if !strings.Contains(out, "Extracted 2 apps") {
		t.Errorf("expected 'Extracted 2 apps' in summary, got: %q", out)
	}
}

func TestPrinter_Summary_ZeroFiles(t *testing.T) {
	buf := &bytes.Buffer{}
	p := newTestPrinter(buf)
	p.Summary()
	out := buf.String()
	if !strings.Contains(out, "Extracted 0 apps") {
		t.Errorf("expected 'Extracted 0 apps' in zero-file summary, got: %q", out)
	}
}

func TestPrinter_DryRunSuffix(t *testing.T) {
	buf := &bytes.Buffer{}
	p := ui.NewPrinter(buf, false, true)
	p.FileResult(ui.Result{
		Status:  ui.StatusOK,
		SrcPath: "sales.qvw",
		OutDir:  "sales.qvw",
		Files:   []string{"script.qvs"},
	})
	out := buf.String()
	if !strings.Contains(out, "[dry run]") {
		t.Errorf("expected '[dry run]' suffix in dry-run output, got: %q", out)
	}
}

func TestPrinter_Summary_DryRun(t *testing.T) {
	buf := &bytes.Buffer{}
	p := ui.NewPrinter(buf, false, true)
	p.FileResult(ui.Result{Status: ui.StatusOK, SrcPath: "a.qvw", OutDir: "a.qvw", Files: []string{"script.qvs"}})
	p.FileResult(ui.Result{Status: ui.StatusWarn, SrcPath: "b.qvw", Message: "no script found"})
	p.Summary()
	out := buf.String()
	if !strings.Contains(out, "Dry run — 2 files would be extracted") {
		t.Errorf("expected 'Dry run — 2 files would be extracted' in summary, got: %q", out)
	}
}
```

Keep `TestPrinter_WarnLine`, `TestPrinter_ErrLine`, and `TestClearSpinner_PaddingMatchesSpinnerTextWidth` unchanged.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test ./internal/ui/... -run "TestPrinter" -v
```

Expected: FAIL — compile error for `QVSPath`/`CharCount`/undefined fields, plus `"Extracted N scripts"` assertion failures

- [ ] **Step 3: Update `Result`, `printOK`, and `Summary` in `internal/ui/output.go`**

Replace the `Result` struct:

```go
// Result holds all information about a single file processing outcome.
type Result struct {
	Status  Status
	SrcPath string
	OutDir  string   // output folder path, no trailing slash
	Files   []string // artifact filenames written, in canonical order
	Message string
}
```

Replace `printOK`:

```go
func (p *Printer) printOK(r Result) {
	sym := p.colorize("✓", okStyle)
	fileList := strings.Join(r.Files, ", ")
	line := fmt.Sprintf("  %s  %s → %s/  (%s)", sym, r.SrcPath, r.OutDir, fileList)
	if p.dryRun {
		line += "  " + p.colorize("[dry run]", dimStyle)
	}
	_, _ = fmt.Fprintln(p.w, line)
}
```

Replace the summary line in `Summary`:

```go
line = fmt.Sprintf("  Extracted %d apps   %s", p.okCount, counts)
```

Remove `formatCount` (no longer used).

- [ ] **Step 4: Run the tests to verify they pass**

```bash
go test ./internal/ui/... -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/output.go internal/ui/output_test.go
git commit -m "feat: update Result struct and Printer for folder-based output"
git push
```

---

## Chunk 4: CLI — Flag semantics and `extract` command wiring

### Task 4: Rewrite flag logic and wire new extractors in `cmd/extract.go`

**Files:**
- Modify: `cmd/extract.go`
- Modify: `cmd/extract_test.go`

**Context:** Cobra flag semantics change from `--script` defaulting `true` (error when false) to four flags all defaulting `false` with "no Changed flags = all" logic. The extraction loop switches from `ExtractScriptFromQVF`/`ExtractScript` + `WriteScript` to `ParseQVF`/`ExtractScript` + `WriteArtifacts`. The `Result` struct passed to the printer changes from `QVSPath`/`CharCount` to `OutDir`/`Files`.

**Selection logic:**

```
anyChanged := cmd.Flags().Changed("script") || cmd.Flags().Changed("measures") ||
              cmd.Flags().Changed("dimensions") || cmd.Flags().Changed("variables")
extractAll := !anyChanged
doScript     := extractAll || script
doMeasures   := extractAll || measures
doDimensions := extractAll || dimensions
doVariables  := extractAll || variables

if anyChanged && !doScript && !doMeasures && !doDimensions && !doVariables {
    // emit error: no artifact type selected
}
```

**Building the artifacts slice (canonical order):**

For each successfully extracted file:
1. If `doScript` and script content is not empty (or file is QVW): add `Artifact{Name: "script.qvs", Content: []byte(scriptContent)}`
2. If `doMeasures` and file is `.qvf`: add `Artifact{Name: "measures.json", Content: marshaledMeasures}`
3. If `doDimensions` and file is `.qvf`: add `Artifact{Name: "dimensions.json", Content: marshaledDimensions}`
4. If `doVariables` and file is `.qvf`: add `Artifact{Name: "variables.json", Content: marshaledVariables}`

For QVW files, only `script.qvs` is ever added (the JSON artifact flags are silently skipped).

**Empty artifact lists:** JSON files are always written for QVF files when their flag is active, even if the list is empty — use `json.MarshalIndent([]T{}, "", "  ")` which produces `[]`.

**NoScriptError for QVF:** After switching to `ParseQVF`, a QVF file with no script will not produce a `NoScriptError` from `ParseQVF`. The "no script found" warn path only triggers if no artifacts would be written at all (i.e. `doScript` is active but script is empty AND no JSON artifacts were requested). The spec says warn behavior triggers when the file produces nothing to write. Since JSON files are always written for QVF even if empty, a QVF file will always have something to write when any non-script flag is active. The logic simplifies to: if `len(artifacts) == 0`, emit warn "no script found".

**RelOut computation for `OutDir`:** Same existing logic pattern but now the value is `outDir` value (a folder path), not a file path. Relativize `resolvedOutDir` against `sourceDir` in alongside mode, or against `outDir` in mirror mode.

- [ ] **Step 1: Update affected tests in `cmd/extract_test.go`**

Update `TestExtractCmd_Integration_ValidFixture` — it currently reads `filepath.Join(outDir, "valid.qvs")`. Change to read from the folder:

```go
gotBytes, readErr := os.ReadFile(filepath.Join(outDir, "valid.qvw", "script.qvs"))
```

Note: the golden file stays at `internal/extractor/testdata/fixtures/valid.qvs.golden` — the content is identical, only the output path changed. The `goldenPath` variable in the test keeps pointing to that file unchanged.

Update `TestExtractCmd_DryRunNoFilesWritten` — currently checks for `.qvs` extension. Change to assert no entries at all:

```go
entries, _ := os.ReadDir(outDir)
if len(entries) != 0 {
    t.Errorf("expected no output in dry-run, found %d entries", len(entries))
}
```

`TestExtractCmd_NoArtifactSelected` passes `--script=false`. It must continue to work after the flag change. Verify the test still passes with the new "all false = error" logic (it should, since `--script` is `Changed` and equals `false`, with no other flags set).

- [ ] **Step 2: Run the existing tests to see current state**

```bash
go test ./cmd/... -v 2>&1 | head -60
```

Note which tests pass and which fail (they will mostly compile-fail once we start editing).

- [ ] **Step 3: Rewrite `cmd/extract.go`**

Replace the entire file:

```go
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/mattiasthalen/qlik-parser/internal/extractor"
	"github.com/mattiasthalen/qlik-parser/internal/ui"
)

func newExtractCmd() *cobra.Command {
	var sourceDir string
	var outDir string
	var dryRun bool
	var script bool
	var measures bool
	var dimensions bool
	var variables bool

	cmd := &cobra.Command{
		Use:   "extract",
		Short: "Extract artifacts from .qvw and .qvf files",
		Long: `Recursively scans --source for .qvw and .qvf files and extracts embedded
artifacts to a per-file folder alongside or under --out.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			anyChanged := cmd.Flags().Changed("script") ||
				cmd.Flags().Changed("measures") ||
				cmd.Flags().Changed("dimensions") ||
				cmd.Flags().Changed("variables")
			extractAll := !anyChanged
			doScript     := extractAll || script
			doMeasures   := extractAll || measures
			doDimensions := extractAll || dimensions
			doVariables  := extractAll || variables

			if anyChanged && !doScript && !doMeasures && !doDimensions && !doVariables {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "error: no artifact type selected\n")
				return ExitError(1)
			}

			if sourceDir == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("could not determine working directory: %w", err)
				}
				sourceDir = cwd
			}

			info, err := os.Stat(sourceDir)
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "error: --source %q: %v\n", sourceDir, err)
				return ExitError(1)
			}
			if !info.IsDir() {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "error: --source %q is a file, not a directory\n", sourceDir)
				return ExitError(1)
			}

			if outDir != "" {
				if err := os.MkdirAll(outDir, 0755); err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "error: cannot create --out directory %q: %v\n", outDir, err)
					return ExitError(1)
				}
			}

			qlikPaths, walkWarns := extractor.Walk(sourceDir)
			for _, w := range walkWarns {
				log.Warn().Msg(w)
			}

			isTTY := ui.IsTTY(os.Stdout)
			printer := ui.NewPrinter(cmd.OutOrStdout(), isTTY, dryRun)

			hasErr := false

			for i, srcPath := range qlikPaths {
				printer.UpdateSpinner(i+1, len(qlikPaths))

				relPath, err := filepath.Rel(sourceDir, srcPath)
				if err != nil {
					relPath = filepath.Base(srcPath)
				}

				isQVF := filepath.Ext(srcPath) == ".qvf"

				// Build the artifact slice.
				var artifacts []extractor.Artifact

				if isQVF {
					qvfData, parseErr := extractor.ParseQVF(srcPath)
					if parseErr != nil {
						hasErr = true
						printer.ClearSpinner()
						errMsg := parseErr.Error()
						if after, ok := strings.CutPrefix(errMsg, srcPath+": "); ok {
							errMsg = after
						}
						printer.FileResult(ui.Result{
							Status:  ui.StatusErr,
							SrcPath: relPath,
							Message: errMsg,
						})
						continue
					}

					if doScript && qvfData.Script != "" {
						artifacts = append(artifacts, extractor.Artifact{
							Name:    "script.qvs",
							Content: []byte(qvfData.Script),
						})
					}
					if doMeasures {
						b, _ := json.MarshalIndent(qvfData.Measures, "", "  ")
						artifacts = append(artifacts, extractor.Artifact{Name: "measures.json", Content: b})
					}
					if doDimensions {
						b, _ := json.MarshalIndent(qvfData.Dimensions, "", "  ")
						artifacts = append(artifacts, extractor.Artifact{Name: "dimensions.json", Content: b})
					}
					if doVariables {
						b, _ := json.MarshalIndent(qvfData.Variables, "", "  ")
						artifacts = append(artifacts, extractor.Artifact{Name: "variables.json", Content: b})
					}
				} else {
					// QVW: script only
					if doScript {
						scriptContent, extractErr := extractor.ExtractScript(srcPath)
						if extractErr != nil {
							var noScript *extractor.NoScriptError
							if errors.As(extractErr, &noScript) {
								printer.ClearSpinner()
								printer.FileResult(ui.Result{
									Status:  ui.StatusWarn,
									SrcPath: relPath,
									Message: "no script found",
								})
								continue
							}
							hasErr = true
							printer.ClearSpinner()
							errMsg := extractErr.Error()
							if after, ok := strings.CutPrefix(errMsg, srcPath+": "); ok {
								errMsg = after
							}
							printer.FileResult(ui.Result{
								Status:  ui.StatusErr,
								SrcPath: relPath,
								Message: errMsg,
							})
							continue
						}
						artifacts = append(artifacts, extractor.Artifact{
							Name:    "script.qvs",
							Content: []byte(scriptContent),
						})
					}
				}

				if len(artifacts) == 0 {
					printer.ClearSpinner()
					printer.FileResult(ui.Result{
						Status:  ui.StatusWarn,
						SrcPath: relPath,
						Message: "no script found",
					})
					continue
				}

				resolvedOutDir := extractor.ResolveOutputDir(srcPath, sourceDir, outDir)
				relOut, err := filepath.Rel(sourceDir, resolvedOutDir)
				if err != nil {
					relOut = filepath.Base(resolvedOutDir)
				}
				if outDir != "" && outDir != sourceDir {
					if r, err := filepath.Rel(outDir, resolvedOutDir); err == nil {
						relOut = r
					}
				}

				fileNames := make([]string, len(artifacts))
				for j, a := range artifacts {
					fileNames[j] = a.Name
				}

				writeErr := extractor.WriteArtifacts(resolvedOutDir, artifacts, dryRun)
				if writeErr != nil {
					hasErr = true
					printer.ClearSpinner()
					printer.FileResult(ui.Result{
						Status:  ui.StatusErr,
						SrcPath: relPath,
						Message: writeErr.Error(),
					})
					continue
				}

				printer.ClearSpinner()
				printer.FileResult(ui.Result{
					Status:  ui.StatusOK,
					SrcPath: relPath,
					OutDir:  relOut,
					Files:   fileNames,
				})
			}

			printer.Summary()

			if hasErr {
				return ExitError(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&sourceDir, "source", "s", "", "Source directory to scan for .qvw and .qvf files (default: current directory)")
	cmd.Flags().StringVarP(&outDir, "out", "o", "", "Export directory (default: alongside source files)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be extracted without writing files")
	cmd.Flags().BoolVar(&script, "script", false, "Extract load scripts")
	cmd.Flags().BoolVar(&measures, "measures", false, "Extract master measures (QVF only)")
	cmd.Flags().BoolVar(&dimensions, "dimensions", false, "Extract master dimensions (QVF only)")
	cmd.Flags().BoolVar(&variables, "variables", false, "Extract variables (QVF only)")

	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		_, _ = fmt.Fprintf(c.ErrOrStderr(), "error: %v\n", err)
		return ExitError(2)
	})

	return cmd
}

// ExitCodeError signals a specific exit code to main.
type ExitCodeError struct {
	Code int
}

func (e *ExitCodeError) Error() string {
	return fmt.Sprintf("exit %d", e.Code)
}

// ExitError creates an ExitCodeError.
func ExitError(code int) error {
	return &ExitCodeError{Code: code}
}
```

- [ ] **Step 4: Run cmd tests to verify they pass**

```bash
go test ./cmd/... -v
```

Expected: all PASS

- [ ] **Step 5: Run the full test suite**

```bash
go test ./... -v 2>&1 | tail -30
```

Expected: all PASS (integration tests skipped if fixtures absent)

- [ ] **Step 6: Commit**

```bash
git add cmd/extract.go cmd/extract_test.go
git commit -m "feat: rewrite extract command with multi-artifact flag semantics and folder output"
git push
```

---

## Chunk 5: Integration tests and golden files

### Task 5: Update integration tests and create golden files

**Files:**
- Modify: `internal/extractor/qlikview_integration_test.go`
- Create: `internal/extractor/testdata/fixtures/measures.json.golden`
- Create: `internal/extractor/testdata/fixtures/dimensions.json.golden`
- Create: `internal/extractor/testdata/fixtures/variables.json.golden`

**Context:** The integration tests use real fixtures (`Qlik_Sense_Content_Monitor.qvf` and `Governance.Dashboard.2.1.4.qvw`). The test assertions must be updated for the new summary wording and new output paths. New tests assert QVF artifact counts and spot-check known values.

- [ ] **Step 1: Write new integration test cases**

Add to `internal/extractor/qlikview_integration_test.go`:

```go
func TestQlikview_ParseQVF_ArtifactCounts(t *testing.T) {
	skipIfNoQlikviewFixtures(t)

	path := filepath.Join(qlikviewTestdata, "Qlik_Sense_Content_Monitor.qvf")
	data, err := extractor.ParseQVF(path)
	if err != nil {
		t.Fatalf("ParseQVF failed: %v", err)
	}

	if len(data.Measures) != 171 {
		t.Errorf("expected 171 measures, got %d", len(data.Measures))
	}
	if len(data.Dimensions) != 78 {
		t.Errorf("expected 78 dimensions, got %d", len(data.Dimensions))
	}
	if len(data.Variables) != 43 {
		t.Errorf("expected 43 variables, got %d", len(data.Variables))
	}
}

func TestQlikview_ParseQVF_SpotCheckMeasure(t *testing.T) {
	skipIfNoQlikviewFixtures(t)

	path := filepath.Join(qlikviewTestdata, "Qlik_Sense_Content_Monitor.qvf")
	data, err := extractor.ParseQVF(path)
	if err != nil {
		t.Fatalf("ParseQVF failed: %v", err)
	}

	var sessions *extractor.Measure
	for i := range data.Measures {
		if data.Measures[i].Label == "Sessions" {
			sessions = &data.Measures[i]
			break
		}
	}
	if sessions == nil {
		t.Fatal("measure with label 'Sessions' not found")
	}
	if sessions.Def != "Count(DISTINCT SessionID)" {
		t.Errorf("Sessions.Def = %q, want %q", sessions.Def, "Count(DISTINCT SessionID)")
	}
}

func TestQlikview_QVW_NoJSONArtifacts(t *testing.T) {
	skipIfNoQlikviewFixtures(t)

	srcDir, _ := filepath.Abs(qlikviewTestdata)
	outDir := t.TempDir()

	root := cmd.NewRootCmd()
	root.SetArgs([]string{"extract", "--source", srcDir, "--out", outDir})
	buf := &bytes.Buffer{}
	root.SetOut(buf)

	if err := root.Execute(); err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	// The QVW output folder should contain only script.qvs, no JSON files.
	qvwOutDir := filepath.Join(outDir, "Governance.Dashboard.2.1.4.qvw")
	entries, err := os.ReadDir(qvwOutDir)
	if err != nil {
		t.Fatalf("output dir not found: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			t.Errorf("unexpected JSON file in QVW output: %s", e.Name())
		}
	}
	// Must have script.qvs
	scriptPath := filepath.Join(qvwOutDir, "script.qvs")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Errorf("script.qvs not found in QVW output dir: %v", err)
	}
}
```

- [ ] **Step 2: Update existing integration test assertions**

In `TestQlikview_ExtractSucceeds_ExitCode0`, change:

```go
if !strings.Contains(out, "Extracted 2 apps") {
    t.Errorf("expected 'Extracted 2 apps' in summary, got: %q", out)
}
```

In `TestQlikview_AllFilesExtractWithoutError`, the test calls `extractor.ExtractScriptFromQVF` — this still works since that function delegates to `ParseQVF`. No change needed.

- [ ] **Step 3: Run the integration tests to see failures**

```bash
go test ./internal/extractor/... -v -run "TestQlikview" 2>&1 | head -50
```

Expected: `TestQlikview_ExtractSucceeds_ExitCode0` FAIL on `"Extracted 2 apps"`, new tests FAIL if not compiling yet (they should compile fine since they use existing imports)

- [ ] **Step 4: Run the tests to verify the new tests pass**

```bash
go test ./internal/extractor/... -v -run "TestQlikview"
```

Expected: all PASS (artifact counts will only be confirmed if the expected numbers are correct — adjust if needed)

- [ ] **Step 5: Generate golden files by extracting from the real fixture**

Run a quick Go program inline to capture the JSON output (or use the extractor directly):

```bash
cd /workspaces/qlik-parser/.worktrees/feat/artifact-extraction
go run - <<'EOF'
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"github.com/mattiasthalen/qlik-parser/internal/extractor"
)

func main() {
	data, err := extractor.ParseQVF("internal/extractor/testdata/fixtures/integration/Qlik_Sense_Content_Monitor.qvf")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	write := func(name string, v any) {
		b, _ := json.MarshalIndent(v, "", "  ")
		_ = os.WriteFile("internal/extractor/testdata/fixtures/"+name, b, 0644)
		fmt.Printf("wrote %s\n", name)
	}
	write("measures.json.golden", data.Measures)
	write("dimensions.json.golden", data.Dimensions)
	write("variables.json.golden", data.Variables)
}
EOF
```

- [ ] **Step 6: Run the full test suite to confirm everything is green**

```bash
go test ./... -v 2>&1 | grep -E "^(ok|FAIL|---)"
```

Expected: all `ok`, no `FAIL`

- [ ] **Step 7: Commit**

```bash
git add internal/extractor/qlikview_integration_test.go \
        internal/extractor/testdata/fixtures/measures.json.golden \
        internal/extractor/testdata/fixtures/dimensions.json.golden \
        internal/extractor/testdata/fixtures/variables.json.golden
git commit -m "test: update integration tests and add golden files for QVF artifact extraction"
git push
```

---

## Notes for the implementer

- The existing fixture helper in `qvf_test.go` is `buildQVFFixture(t, []byte)`. The plan adds `makeQVFFile(t, map[string]any)` as a convenience wrapper on top of it.
- The artifact count assertions in Task 5 Step 1 (`171`, `78`, `43`) are from the spec. If they don't match the real fixture, adjust and document the correct numbers.
- `json.MarshalIndent` with `""` prefix and `"  "` indent is the canonical output format per the spec.
- Do NOT add `formatCount` back — char count display is gone.
- `TestExtractCmd_Integration_NoScriptIsWarn` should still pass because `no_script.qvw` will produce 0 artifacts (script is not found, no JSON flags active for QVW), triggering the warn path.
- `TestExtractCmd_Integration_ErrorFilesSetExitCode` should still pass because `invalid_zlib.qvw` / `too_short.qvw` will still error.
