# Design: Extract Additional Artifacts (Issue #3)

**Date:** 2026-03-12
**Status:** Approved
**Issue:** [#3 Extract other artefacts](https://github.com/mattiasthalen/qlik-parser/issues/3)

## Overview

Extend `qlik-parser extract` to extract measures, dimensions, and variables from `.qvf` files in addition to load scripts. Output is written to a per-source-file folder. CLI flag semantics change: no flags extracts all artifacts; explicit flags extract only the named ones.

---

## 1. CLI Behavior

### New flags

| Flag | Default | Description |
|---|---|---|
| `--script` | false | Extract load script |
| `--measures` | false | Extract master measures (QVF only) |
| `--dimensions` | false | Extract master dimensions (QVF only) |
| `--variables` | false | Extract variables (QVF only) |

All four flags default to `false` in Cobra. "No flags passed" is detected using `cmd.Flags().Changed("script")` etc. — not by checking the boolean value. If none of the four flags are `Changed`, all four are treated as active.

Passing a flag explicitly as `false` (e.g. `--script=false`) is treated as an error: it would mean the user explicitly deselected an artifact without selecting any other. The validation guard checks: if any flag is `Changed` and no flag resolves to `true`, emit `"no artifact type selected"` (exit code 1). This preserves the behavior tested by `TestExtractCmd_NoArtifactSelected`.

### Selection logic

- **No artifact flags passed** (`Changed` returns false for all) → extract all artifacts
- **Any artifact flag(s) passed as `true`** → extract only the named ones
- **All explicitly passed flags are `false`** → error: no artifact type selected

```
qlik-parser extract                          # all artifacts
qlik-parser extract --script --measures      # script + measures only
qlik-parser extract --script                 # script only
qlik-parser extract --script=false           # error: no artifact type selected
```

### Breaking change

Previously `--script` defaulted to `true`. Under the new semantics, passing no flags extracts *all* artifacts. Existing explicit `--script` invocations are unaffected.

---

## 2. Output Layout

Each source file gets its own output folder named `<filename>.<ext>/`.

### Alongside mode (no `--out`)

```
/data/
  sales.qvf  →  /data/sales.qvf/
                   script.qvs
                   measures.json
                   dimensions.json
                   variables.json

  report.qvw  →  /data/report.qvw/
                    script.qvs
```

### Mirror mode (`--out /export`)

```
/data/etl/sales.qvf  →  /export/etl/sales.qvf/
                           script.qvs
                           measures.json
                           ...
```

### Collision avoidance

Since the folder is named after the full filename including extension (`sales.qvf/` vs `sales.qvw/`), a directory containing both `sales.qvf` and `sales.qvw` produces two separate output folders with no collision.

### QVW behavior

QVW files only support `script.qvs`. If `--measures`, `--dimensions`, or `--variables` are active (explicitly or via the "all" default), those artifact files are not written for QVW sources — not an error, no warning. The result line reports only the files actually written.

### Empty artifact lists

If a QVF file contains no measures, dimensions, or variables, the corresponding JSON file is still written as an empty array `[]`. This makes downstream consumers predictable.

### Dry-run mode

In dry-run mode, the output folder and files are not written. The result line behaves identically to non-dry-run but appends `[dry run]`. All three outcome symbols (✓ ⚠ ✗) apply in dry-run mode as they do normally.

```
✓  sales.qvf → sales.qvf/  (script.qvs, measures.json, dimensions.json, variables.json) [dry run]
⚠  empty.qvw → empty.qvw/  no script found  [dry run]
```

---

## 3. Exporter Changes

### `ResolveOutputPath` → `ResolveOutputDir`

The existing `ResolveOutputPath(inputPath, sourceDir, outDir string) string` (which returns a `.qvs` file path) is replaced by `ResolveOutputDir(inputPath, sourceDir, outDir string) string`, which returns the output folder path without a trailing slash (standard Go path convention — the trailing slash is added by the UI layer when displaying `OutDir`).

The relativization rule follows the same logic as the current function:
- **Alongside mode** (`outDir` empty or equals `sourceDir`): folder placed next to the source file, relative to `sourceDir`
- **Mirror mode** (`outDir` differs): folder mirrors the directory tree under `outDir`

`ResolveOutputPath` is removed. Its existing tests (`TestResolveOutputPath_*`) are updated to test `ResolveOutputDir` with folder paths as expected values.

### `WriteScript` → `WriteArtifacts`

The existing `WriteScript(outPath, script string, dryRun bool) error` is replaced by:

```go
func WriteArtifacts(outDir string, artifacts []Artifact, dryRun bool) error
```

where:

```go
type Artifact struct {
    Name    string // e.g. "script.qvs", "measures.json"
    Content []byte
}
```

Using a slice (not a map) ensures deterministic write order and predictable error reporting. The canonical order is: `script.qvs`, `measures.json`, `dimensions.json`, `variables.json`. `WriteArtifacts` is **fail-fast**: it returns the first error encountered and does not attempt remaining writes.

In dry-run mode, the function returns nil without creating the directory or writing any files.

`WriteScript` is removed.

---

## 4. QVF Parser Refactor

### Approach: single-pass parser

`internal/extractor/qvf.go` introduces `ParseQVF(path string) (*QVFData, error)` which scans all zlib blocks once and returns all artifacts. The `extract` command decides which fields to write based on resolved flags.

### Data types

```go
type QVFData struct {
    Script     string
    Measures   []Measure
    Dimensions []Dimension
    Variables  []Variable
}

type Measure struct {
    ID          string   `json:"id"`
    Label       string   `json:"label"`
    Def         string   `json:"def"`
    Tags        []string `json:"tags"`
    Description string   `json:"description"`
}

type Dimension struct {
    ID          string   `json:"id"`
    Label       string   `json:"label"`
    Fields      []string `json:"fields"`
    Tags        []string `json:"tags"`
    Description string   `json:"description"`
}

type Variable struct {
    ID      string          `json:"id"`
    Name    string          `json:"name"`
    Comment string          `json:"comment"`
    Value   json.RawMessage `json:"value"` // passed through verbatim from source JSON
}
```

`Variable.Value` is `json.RawMessage` and passed through verbatim — a number stays a number (`180`), a string stays a string (`"NaN"`). No re-encoding is applied.

### Block detection logic

Each decompressed zlib block is null-terminated and inspected:

| Pattern | Action |
|---|---|
| Top-level `qScript` key | → populate `Script` (first match wins; subsequent blocks with `qScript` are ignored) |
| Top-level `qInfo.qType == "measure"` | → append to `Measures` |
| Top-level `qInfo.qType == "dimension"` | → append to `Dimensions` |
| Top-level `qId == "user_variablelist"` | → populate `Variables` from `qEntryList` |

### Backwards compatibility

The existing `ExtractScriptFromQVF(path string) (string, error)` is kept and internally delegates to `ParseQVF`, returning only `QVFData.Script`. The existing unit tests in `qvf_test.go` remain valid and are supplemented by new tests for `ParseQVF` covering all four artifact types.

---

## 5. JSON Output Format

### measures.json

```json
[
  {
    "id": "EHphN",
    "label": "Sessions",
    "def": "Count(DISTINCT SessionID)",
    "tags": [],
    "description": ""
  }
]
```

### dimensions.json

```json
[
  {
    "id": "xdLfma",
    "label": "Application",
    "fields": ["AppName"],
    "tags": [],
    "description": ""
  }
]
```

### variables.json

```json
[
  {
    "id": "cbb1fc8b-3365-4e57-98f6-2684761f4af2",
    "name": "vSessionLogDaysBack",
    "comment": "",
    "value": 180
  }
]
```

JSON is written with `json.MarshalIndent` (2-space indent) for human readability.

---

## 6. Result and UI Changes

### Result struct

`QVSPath` and `CharCount` are removed. `OutDir` and `Files` are added:

```go
type Result struct {
    Status  Status
    SrcPath string
    OutDir  string   // output folder path, no trailing slash, relativized same as current QVSPath
    Files   []string // artifact filenames written, in canonical order
    Message string
}
```

`OutDir` relativization follows the same rule as the current `relOut` in `cmd/extract.go`: relative to `sourceDir` in alongside mode, relative to `outDir` in mirror mode.

`Files` is always in canonical order: `script.qvs`, `measures.json`, `dimensions.json`, `variables.json` (only files actually written are included). The UI layer appends a trailing slash to `OutDir` when formatting for display.

### Terminal output

```
✓  sales.qvf → sales.qvf/  (script.qvs, measures.json, dimensions.json, variables.json)
⚠  empty.qvw → empty.qvw/  no script found
✗  corrupt.qvf              zlib: invalid header
```

### Summary line

```
Extracted 10 apps  ✓ 10  ⚠ 1  ✗ 1
```

---

## 7. Testing

### Tests requiring updates

The following existing tests break under the new design and must be updated:

| Test | File | Change required |
|---|---|---|
| `TestExtractCmd_NoArtifactSelected` | `cmd/extract_test.go` | Still valid — passes `--script=false`, expects exit 1. No change needed. |
| `TestPrinter_SuccessLine` | `internal/ui/output_test.go` | Remove char count assertion; assert file list in output instead. |
| `TestResolveOutputPath_*` | `internal/extractor/exporter_test.go` | Rename to `TestResolveOutputDir_*`; expected values change to folder paths. |
| `TestExtractCmd_Integration_ValidFixture` | `cmd/extract_test.go` | Read from `outDir/valid.qvw/script.qvs` instead of `outDir/valid.qvs`. |
| `TestExtractCmd_DryRunNoFilesWritten` | `cmd/extract_test.go` | Currently checks `filepath.Ext(e.Name()) == ".qvs"`; change to assert `len(entries) == 0` (no output folder created). |
| Integration test summary assertion | `internal/extractor/qlikview_integration_test.go` | Change `"Extracted 2 scripts"` → `"Extracted 2 apps"`. |

### New unit tests

- `ParseQVF`: table-driven tests using existing synthetic fixtures for script; new synthetic fixtures for measures, dimensions, variables
- `WriteArtifacts`: alongside mode, mirror mode, dry-run, directory creation, fail-fast error
- Flag resolution logic: "no flags → all", "some flags → those only", "all false → error"

### Integration tests

- Real fixture `Qlik_Sense_Content_Monitor.qvf`: assert counts (171 measures, 78 dimensions, 43 variables) and spot-check known values (e.g. measure label `"Sessions"` with def `"Count(DISTINCT SessionID)"`)
- Real fixture `Governance.Dashboard.2.1.4.qvw`: assert only `script.qvs` is written when all flags active; no JSON files present in output folder

### Golden files

Located at `internal/extractor/testdata/fixtures/` alongside existing golden files. Hand-authored:
- `measures.json.golden`
- `dimensions.json.golden`
- `variables.json.golden`

---

## 8. Out of Scope

- QVW measures/dimensions/variables (binary format, not worthwhile without spec)
- Data connections (not embedded in `.qvf`)
- Chart/sheet objects (deferred to a future issue)
- Variable source definitions (the `Set` expression in the script) — `qValue` is the runtime-computed value; the definition lives in the script text
