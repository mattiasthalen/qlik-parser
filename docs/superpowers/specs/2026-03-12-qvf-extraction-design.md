# QVF Script Extraction — Design Spec

**Date:** 2026-03-12
**Branch:** feat/qvf-extraction

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

---

## Design

### 1. New file: `internal/extractor/qvf.go`

Exports a single function:

```go
func ExtractScriptFromQVF(path string) (string, error)
```

**Algorithm:**
1. Read the entire file into memory.
2. Scan byte-by-byte for zlib stream candidates: bytes `0x78` followed by `0x01`, `0x9C`, `0xDA`, or `0x5E`.
3. For each candidate, attempt `zlib.Decompress`. On success, attempt `json.Unmarshal` into a struct with a `QScript string` field.
4. Return the first successfully unmarshalled `QScript` value.
5. If no stream yields a `qScript`, return `&NoScriptError{Path: path}` — reusing the existing error type for consistent caller behaviour.

Error cases mirror QVW:
- File unreadable → wrapped `os` error
- No zlib stream with `qScript` → `*NoScriptError`

### 2. Modified: `internal/extractor/walker.go`

Extend `Walk` to collect both `.qvw` and `.qvf` files:

```go
if !d.IsDir() && (filepath.Ext(path) == ".qvw" || filepath.Ext(path) == ".qvf") {
    paths = append(paths, path)
}
```

No changes to the function signature or return types.

### 3. Modified: `internal/extractor/exporter.go`

`ResolveOutputPath` currently hardcodes `.qvw` suffix trimming. Generalise to strip the actual extension:

```go
base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
```

Output extension by format:
- `.qvw` → `.qvs`
- `.qvf` → `.qvf.qvs`

This avoids silent collision when `report.qvw` and `report.qvf` exist in the same directory. The function signature gains an `ext string` parameter (or derives it from path).

### 4. Modified: `cmd/extract.go`

After `Walk`, dispatch on file extension:

```go
switch filepath.Ext(path) {
case ".qvw":
    scriptContent, extractErr = extractor.ExtractScript(path)
case ".qvf":
    scriptContent, extractErr = extractor.ExtractScriptFromQVF(path)
}
```

`ResolveOutputPath` call updated to handle the new output extension logic.

---

## Output Extension Convention

| Input | Output |
|-------|--------|
| `report.qvw` | `report.qvs` |
| `report.qvf` | `report.qvf.qvs` |

Rationale: avoids silent overwrite when both formats share a filename in the same directory. Existing QVW workflows are unaffected.

---

## Script Marker Convention

Both formats use `///$tab` as the script start marker. All integration tests (QVW and QVF) assert `strings.HasPrefix(script, "///$tab")` — not just `"///"`. The existing QVW integration test is updated to match.

---

## Testing

### Unit tests: `internal/extractor/qvf_test.go`

| Test | Fixture | Expected |
|------|---------|----------|
| File too short | synthetic | error (not NoScriptError) |
| No zlib stream with qScript | synthetic | `*NoScriptError` |
| Valid qScript in zlib JSON | synthetic minimal | script string returned |

Synthetic fixtures are constructed in-memory (no files on disk) using the same pattern as `qvw_test.go`.

### Integration tests: `internal/extractor/qlikview_integration_test.go`

Updates:
- `TestQlikview_WalkerFindsAllFiles`: expect **2** files (1 QVW + 1 QVF)
- `TestQlikview_AllFilesExtractWithoutError`: covers both via the dispatch pattern
- `TestQlikview_AllScriptsStartWithTripleSlash`: tightened to `///$tab` for both formats
- `TestQlikview_ExtractSucceeds_ExitCode0`: expect `"Extracted 2 scripts"` in summary

---

## What Is Not Changing

- The `NoScriptError` type and `IsNoScript` helper — reused as-is
- The `WriteScript` function — unchanged
- The `--script`, `--dry-run`, `--source`, `--out` CLI flags — unchanged
- TTY/spinner/printer logic — unchanged
