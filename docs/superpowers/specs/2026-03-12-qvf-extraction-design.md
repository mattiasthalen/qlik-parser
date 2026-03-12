# QVF Script Extraction — Design Spec

**Date:** 2026-03-12
**Branch:** feat/qvf-extraction
**Closes:** [#2 — Add support for qvf](https://github.com/mattiasthalen/qlik-parser/issues/2)

---

## Problem

The tool currently extracts load scripts from `.qvw` (QlikView) files only. Qlik Sense apps are stored as `.qvf` files, which have a completely different binary format. Users with Qlik Sense apps cannot use the tool today.

---

## Format Overview

### QVW (existing)
- Fixed 23-byte binary header
- Single zlib-compressed block immediately following the header
- Decompressed blob contains the load script delimited by `///$tab` at the start and `\n\x00\x00+` at the end

### QVF (new)
- Proprietary binary container with an unknown header structure
- Multiple zlib-compressed blocks at various offsets throughout the file
- The load script lives inside one block whose decompressed content is a JSON object: `{"qScript": "...load script..."}`
- The script itself starts with `///$tab` (same convention as QVW)
- Files can be large (Qlik Sense apps embed data models, visualisations, etc.); the design accepts reading the whole file into memory, which is consistent with the existing QVW approach

---

## Design

### 1. New file: `internal/extractor/qvf.go`

Exports a single function:

```go
func ExtractScriptFromQVF(path string) (string, error)
```

**Algorithm:**
1. Read the entire file into memory.
2. Scan byte-by-byte for zlib stream candidates: byte `0x78` followed by one of `0x01`, `0x5E`, `0x9C`, `0xDA` (the four standard RFC 1950 level values for CMF=0x78).
3. For each candidate, attempt `zlib.Decompress`. On failure (not a real zlib stream), **silently skip** and continue scanning.
4. On successful decompression, attempt `json.Unmarshal` into a struct with a `QScript string \`json:"qScript"\`` field. On failure (not the target block), **silently skip**.
5. Return the first successfully unmarshalled non-empty `QScript` value.
6. If no stream yields a `qScript`, return `&NoScriptError{Path: path}` — reusing the existing error type. The condition for QVF is "no block with a valid `qScript` field" rather than "no `///` marker", but `NoScriptError.Error()` returns the generic `"no script found"` which is accurate for both formats.

Error cases mirror QVW:
- File unreadable → wrapped `os` error
- No zlib stream with `qScript` → `*NoScriptError`

### 2. Modified: `internal/extractor/walker.go`

Extend `Walk` to collect both `.qvw` and `.qvf` files:

```go
ext := filepath.Ext(path)
if !d.IsDir() && (ext == ".qvw" || ext == ".qvf") {
    paths = append(paths, path)
}
```

No changes to the function signature or return types.

**Test impact on `walker_test.go`:**
- `TestWalkFindsQVWFiles` is extended (or a parallel `TestWalkFindsQVFFiles` is added) to assert `.qvf` files are also collected.
- `TestWalkIgnoresNonQVW` is renamed to `TestWalkIgnoresUnrelatedExtensions` and its fixture set changes from `{"a.qvf", "b.txt", "c.qvs", "d.QVW"}` to `{"a.txt", "b.qvs", "c.QVW"}` — removing `.qvf` (now collected) and keeping only extensions that must still be ignored. The assertion remains: `len(got) == 0`.

### 3. Modified: `internal/extractor/exporter.go`

`ResolveOutputPath` currently hardcodes `.qvw` suffix trimming. Changes:
- Parameter renamed from `qvwPath` to `inputPath`
- Doc comment updated to reflect the generalized behaviour (not `.qvw`-specific)
- Suffix trimming generalized using `filepath.Ext`

Output extension by format:
- `.qvw` → `.qvs`
- `.qvf` → `.qvf.qvs`

The double-extension for `.qvf` is intentional: it avoids silent overwrite when `report.qvw` and `report.qvf` both exist in the same directory (which would otherwise both resolve to `report.qvs`). It also makes the source format visible in the output filename.

**Test impact:** `exporter_test.go` gains new cases for `.qvf` input → `.qvf.qvs` output. Existing `.qvw` cases continue to pass unchanged.

### 4. Modified: `cmd/extract.go`

After `Walk`, dispatch on file extension:

```go
var scriptContent string
var extractErr error
switch filepath.Ext(path) {
case ".qvw":
    scriptContent, extractErr = extractor.ExtractScript(path)
case ".qvf":
    scriptContent, extractErr = extractor.ExtractScriptFromQVF(path)
}
```

`ResolveOutputPath` call is unchanged in structure; it now correctly derives the output extension from the input extension via `filepath.Ext`.

**Help text:** The `--source` flag description and `Short`/`Long` command descriptions are updated to reference both `.qvw` and `.qvf` files instead of just `.qvw`.

### 5. Modified: `internal/ui/output.go`

`Result.QVWPath` is renamed to `Result.SrcPath` to reflect that it now holds paths for both `.qvw` and `.qvf` inputs. All callers (`cmd/extract.go` — all five `ui.Result{...}` literals, `output_test.go` — all fixture literals) are updated accordingly.

---

## Output Extension Convention

| Input | Output |
|-------|--------|
| `report.qvw` | `report.qvs` |
| `report.qvf` | `report.qvf.qvs` |

Rationale: avoids silent overwrite when both formats share a filename in the same directory. Existing QVW workflows are unaffected.

---

## Script Marker Convention

Both formats use `///$tab` as the script start marker. All tests that check the script prefix are updated to assert `strings.HasPrefix(script, "///$tab")` — both unit tests (`qvw_test.go` — all prefix assertions including `TestExtractScript_ValidFile` and `TestExtractScript_NoEndMarker`) and integration tests.

---

## Test Fixtures

The integration fixture directory `testdata/fixtures/integration/` already contains:
- `Governance.Dashboard.2.1.4.qvw` (existing)
- `Qlik_Sense_Content_Monitor.qvf` (already present in the repo)

No new fixture files need to be added. The integration tests will use both existing files.

---

## Testing

### Unit tests: `internal/extractor/qvf_test.go`

| Test | Fixture | Expected |
|------|---------|----------|
| File too short / unreadable | synthetic in-memory | wrapped `os` error |
| Valid zlib streams but none with `qScript` | synthetic in-memory | `*NoScriptError` |
| Valid `qScript` in zlib-compressed JSON | synthetic in-memory | script string returned |

Synthetic fixtures are constructed in-memory (no files on disk), consistent with the pattern in `qvw_test.go`.

### Unit tests: `internal/extractor/qvw_test.go`

- All script prefix assertions tightened from `"///"` to `"///$tab"` (`TestExtractScript_ValidFile` and `TestExtractScript_NoEndMarker`).

### Unit tests: `internal/extractor/exporter_test.go`

- New cases for `.qvf` input → `.qvf.qvs` output.
- Existing `.qvw` cases unchanged.

### Unit tests: `internal/extractor/walker_test.go`

- `TestWalkIgnoresNonQVW` renamed to `TestWalkIgnoresUnrelatedExtensions`; `.qvf` removed from its fixture set.
- `TestWalkFindsQVFFiles` added (parallel to `TestWalkFindsQVWFiles`) asserting `.qvf` files are collected.

### Integration tests: `internal/extractor/qlikview_integration_test.go`

| Test | Change |
|------|--------|
| `TestQlikview_WalkerFindsAllFiles` | Expect **2** files (1 QVW + 1 QVF) |
| `TestQlikview_AllFilesExtractWithoutError` | Updated to dispatch by extension (`switch filepath.Ext(p)`) calling the appropriate extractor |
| `TestQlikview_AllScriptsStartWithTripleSlash` | Updated to dispatch by extension; assertion tightened to `"///$tab"` |
| `TestQlikview_ExtractSucceeds_ExitCode0` | Expect `"Extracted 2 scripts"` in summary |

The `skipIfNoQlikviewFixtures` guard remains unchanged — it checks for the directory, which already contains both files.

---

## What Is Not Changing

- The `NoScriptError` type and `IsNoScript` helper — reused as-is
- The `WriteScript` function — unchanged
- The `--script`, `--dry-run`, `--source`, `--out` CLI flags — unchanged
- TTY/spinner/printer logic — unchanged (beyond the `QVWPath` → `SrcPath` rename in `Result`)
