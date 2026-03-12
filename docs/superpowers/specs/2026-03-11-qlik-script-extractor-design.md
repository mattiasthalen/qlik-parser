# QlikView Script Extractor CLI — Design Spec

Date: 2026-03-11

## Overview

A Go CLI tool that extracts QlikView load scripts (`.qvs`) from QVW binary files.

## CLI Interface

Binary name: `qlik-script-extractor`

The CLI provides `version` and root-level `help` commands automatically. We add one command: `export`.

### Subcommand: `export`

```
qlik-script-extractor export [--source <dir>] [--out <dir>] [--dry-run]
```

Flags:
- `--source` / `-s` — source directory to scan for `.qvw` files (default: `os.Getwd()` evaluated when the `export` command's `RunE` function is entered). Must be a directory; passing a single file path is an error. Paths shown in per-file output are relative to `--source` (e.g. `etl/sales.qvw → etl/sales.qvs`).
- `--out` / `-o` — export directory for `.qvs` output (default: `""` — empty string signals alongside mode)
- `--dry-run` — show what would be extracted without writing any files (no short form)

### Output path resolution

- `--out` specified (non-empty): mirror source folder structure under export dir. All intermediate subdirectories are created automatically as needed.
  - e.g. `--source /data --out /out` → `/data/etl/sales.qvw` → `/out/etl/sales.qvs` (creates `/out/etl/` if needed)
- `--out` not specified (empty string): write `.qvs` alongside the `.qvw` file
  - e.g. `/data/etl/sales.qvw` → `/data/etl/sales.qvs`

## Architecture (4-layer pattern)

```
main.go                          Entry — delegates to root command only
cmd/export.go                    Command — flag parsing, calls business logic, exit codes
internal/extractor/
  walker.go                      Recursively finds *.qvw files under source dir
  qvw.go                         Decompresses QVW, extracts script region
  exporter.go                    Resolves output paths, writes .qvs files, dry-run aware
internal/ui/
  output.go                      Terminal output: per-file status, summary (bubbletea/lipgloss)
```

Logging (zerolog) is for debug/structured diagnostics written to stderr. All user-facing output (per-file status lines, spinner, summary) goes through the UI layer to stdout.

## Core Algorithm

### Input validation

Before processing, validate:
- `--source` exists and is a directory; otherwise exit 1 with a clear error message
- File shorter than 23 bytes: emit ERR (`file too short`), skip, continue

### Decompression

Read raw bytes from `.qvw`, skip first 23 bytes, pass remainder to `compress/zlib`. The result is a raw `[]byte` — no UTF-8 conversion at this stage. Conversion happens only after script extraction (see step 7 of Script extraction below).

### Script extraction

All extraction operates on the raw decompressed **byte slice** before any UTF-8 conversion, to avoid null-byte ambiguity after replacement-character substitution. UTF-8 conversion is applied only to the final extracted script bytes before writing to disk.

`///` is the QlikView load script section delimiter. Its position varies per file; scan for first occurrence.

1. Find byte offset of first occurrence of `///` in the decompressed bytes. Let `scriptStart` = that offset (the `///` itself is included as the first three bytes of the extracted script).
2. If not found: emit WARN (`no script found`), skip file, continue
3. Let `region = bytes[scriptStart : min(scriptStart+100_000, len(bytes))]`. Truncation is silent — QlikView scripts are not expected to exceed 100,000 bytes in practice.
4. Search `region` for the **first** occurrence of the end marker: `\r\n` followed by two or more consecutive `\x00` bytes (`\r\n\x00\x00+`), or `\n` followed by two or more consecutive `\x00` bytes (`\n\x00\x00+`)
5. If end marker found: script bytes = `region[:matchStart]` (up to but not including the trailing newline)
6. If no end marker: script bytes = full `region`
7. Convert script bytes to UTF-8 string: `strings.ToValidUTF8(string(scriptBytes), "\uFFFD")`. Replacement characters are preserved as-is in the output. No BOM is written.
8. Write script string as UTF-8 to the output `.qvs` path

### File walking

Recursively walk source directory using `fs.WalkDir`. Collect all `*.qvw` files, sort for deterministic output. Symlinks are not followed. If a subdirectory cannot be read (permission denied): emit WARN for that directory, skip it, continue walking.

## Terminal UI

Built with `bubbletea` + `lipgloss`. Colors and spinner are auto-disabled when stdout is not a TTY (piped/redirected). In non-TTY mode the spinner line is suppressed; only per-file result lines and the summary are printed as plain text.

### During extraction

Spinner with running count: `Extracting... 3/12`. The total count (12) is determined by a full directory walk before extraction begins (two-phase: collect all `.qvw` paths, then process them sequentially).

### Per-file output

```
  ✓  sales.qvw → sales.qvs  (4,821 chars)
  ⚠  empty.qvw  no script found
  ✗  corrupt.qvw  zlib: invalid header
```

Colors: green ✓, yellow ⚠, red ✗.

### Dry-run output

Per-file lines under `--dry-run` use the same symbols as normal output with a `[dry run]` suffix:

```
  ✓  sales.qvw → sales.qvs  (4,821 chars)  [dry run]
  ⚠  empty.qvw  no script found  [dry run]
  ✗  corrupt.qvw  zlib: invalid header  [dry run]
```

### Summary

```
  Extracted 10 scripts   ✓ 10  ⚠ 1  ✗ 1
```

Dry-run summary: `Dry run — 10 files would be extracted  ✓ 10  ⚠ 1  ✗ 1`

## Error Handling & Exit Codes

| Situation | Exit code |
|---|---|
| All files succeeded (warnings are not errors) | 0 |
| No `.qvw` files found | 0 (summary shows 0) |
| Any file had an ERR (incl. dry-run with would-error files) | 1 |
| `--source` does not exist or is not a directory | 1 |
| `--source` is a file, not a directory | 1 |
| `--out` dir cannot be created (e.g. permission denied) | 1 |
| Bad arguments (handled by cobra) | 2 |

Additional rules:
- Per-file errors are non-fatal: log and continue processing remaining files
- `--out` top-level directory is pre-flighted with `os.MkdirAll` before processing begins; if that fails, exit 1 immediately. Mirrored subdirectories are created per-file just before writing; if a subdirectory creation fails for a specific file, it is treated as a per-file ERR (log, continue).
- `--dry-run` with no files found: exit 0, summary shows "Dry run — 0 files would be extracted"
- `--out` equal to `--source` is accepted; `.qvs` files land next to their `.qvw` sources (same as alongside mode)
- `--dry-run` does not suppress error exit codes — files that would fail still count as ERR
- Write failure for a specific `.qvs` file (e.g. disk full, permission denied): treat as per-file ERR (log, continue, exit 1)
- No `///` marker is a WARN (exit 0) because a valid QVW may simply have no load script (e.g. a dashboard-only file). A file < 23 bytes is structurally invalid and cannot be a QVW, hence ERR (exit 1).
- A `--log-level` flag enables debug output via zerolog.

## Testing

Follows >80% coverage requirement. Sequential processing only (no concurrency).

### Unit tests (`internal/extractor/`)

- `qvw_test.go`: decompression and script extraction using fixtures from `testdata/` plus synthetic cases (no `///` marker, file < 23 bytes, invalid zlib, script with no end marker, invalid UTF-8 bytes)
- `walker_test.go`: recursive file discovery with a temp dir tree; verify symlinks not followed; verify unreadable subdir emits warn and continues
- `exporter_test.go`: output path resolution (mirror vs alongside), dry-run (assert no files written), `--out` dir auto-creation

### Integration test (`cmd/export_test.go`)

Run full `export` command against `internal/extractor/testdata/` fixtures, verify `.qvs` files produced with expected content. Golden files (containing only the extracted script string, no metadata) live in `internal/extractor/testdata/` with a `.qvs.golden` extension, mirroring the fixture subdirectory structure. The integration test package registers a `-update` flag: `go test ./cmd/... -update` regenerates golden files.

### Edge cases

| Scenario | Expected behaviour |
|---|---|
| No `///` marker | WARN, skip, continue, exit 0 |
| Invalid zlib data | ERR, skip, continue, exit 1 |
| File < 23 bytes | ERR, skip, continue, exit 1 |
| Empty source dir | Exit 0, summary shows 0 |
| `--out` dir missing | Create dir tree, proceed |
| `--dry-run` with valid files | No files written, exit 0 |
| `--dry-run` with would-error files | No files written, exit 1 |
| `--source` is a file | Exit 1, clear error message |

## Dependencies

- `github.com/spf13/cobra` — subcommand CLI framework
- `github.com/spf13/viper` — config management (not used for `export` flags — no env var or config file overrides for this command)
- `github.com/rs/zerolog` — structured logging (stderr)
- `github.com/charmbracelet/bubbletea` — terminal UI (stdout)
- `github.com/charmbracelet/lipgloss` — terminal styling

Stdlib only for core algorithm: `compress/zlib`, `io/fs`, `path/filepath`, `strings`.
