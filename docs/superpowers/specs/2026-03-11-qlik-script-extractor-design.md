# QlikView Script Extractor CLI — Design Spec

Date: 2026-03-11

## Overview

A Go CLI tool that extracts QlikView load scripts (`.qvs`) from QVW binary files. Built on the `ckeletin-go` skeleton for production-grade scaffolding.

## Bootstrap

- Clone `ckeletin-go` and run `task init name=qlik-script-extractor module=github.com/peiman/qlik-script-extractor`
- Replace current repo contents, preserving `.devcontainer/`, `.claude/`, `docs/`
- Move QVW fixtures from `references/` to `internal/extractor/testdata/`, remove `references/`

## CLI Interface

Binary name: `qlik-script-extractor`

### Subcommand: `export`

```
qlik-script-extractor export [--source <dir>] [--out <dir>] [--dry-run]
```

Flags:
- `--source` / `-s` — source directory to scan for `.qvw` files (default: CWD)
- `--out` / `-o` — export directory for `.qvs` output (default: none — writes alongside source files)
- `--dry-run` — show what would be extracted without writing any files

### Output path resolution

- `--out` specified: mirror source folder structure under export dir
  - e.g. `--source /data --out /out` → `/data/etl/sales.qvw` → `/out/etl/sales.qvs`
- No `--out`: write `.qvs` alongside the `.qvw` file
  - e.g. `/data/etl/sales.qvw` → `/data/etl/sales.qvs`

## Architecture (ckeletin-go 4-layer pattern)

```
main.go                          Entry — delegates to root command only
cmd/export.go                    Command — flag parsing, calls business logic, exit codes
internal/extractor/
  walker.go                      Recursively finds *.qvw files under source dir
  qvw.go                         Decompresses QVW, extracts script region
  exporter.go                    Resolves output paths, writes .qvs files, dry-run aware
internal/ui/
  output.go                      Terminal output: per-file status, summary
```

## Core Algorithm

### Decompression

Read raw bytes from `.qvw`, skip first 23 bytes, decompress remainder with `compress/zlib` (stdlib). Decode as UTF-8 with replacement characters for invalid bytes.

### Script extraction

1. Find first occurrence of `///` in decompressed text
2. If not found: emit WARN, skip file
3. Take up to 100,000 characters from that position
4. Search for end marker: `\r\n` or `\n` followed by 2+ null bytes
5. If end marker found: trim at `start + 2`; otherwise use full 100k region

### File walking

Recursively walk source directory, collect all `*.qvw` files, sort for deterministic output.

## Terminal UI

Built with `bubbletea` + `lipgloss` (provided by ckeletin-go skeleton).

### During extraction

Spinner with running count: `Extracting... 3/12`

### Per-file output

```
  ✓  sales.qvw → sales.qvs  (4,821 chars)
  ⚠  empty.qvw  no script found
  ✗  corrupt.qvw  zlib: invalid header
```

Colors: green ✓, yellow ⚠, red ✗. Auto-disabled when stdout is not a TTY.

### Dry-run output

```
  ~  sales.qvw → sales.qvs  (4,821 chars)  [dry run]
```

### Summary

```
  Extracted 10 scripts   ✓ 10  ⚠ 1  ✗ 1
```

Dry-run summary: `Dry run — 10 files would be extracted`

## Error Handling

- Per-file errors are non-fatal: log and continue
- Exit code 0 if all files succeeded (warnings are not errors)
- Exit code 1 if any file had an ERR
- If `--out` directory does not exist: create it (including parents)
- Empty source directory: clean exit, summary shows 0 files

## Testing

Follows ckeletin-go's >80% coverage requirement.

### Unit tests (`internal/extractor/`)

- `qvw_test.go`: decompression and script extraction against fixtures in `testdata/` plus synthetic cases (no script, truncated file, invalid zlib, script with no end marker)
- `walker_test.go`: recursive file discovery with temp dir tree
- `exporter_test.go`: output path resolution (mirror vs alongside), dry-run (no files written)

### Integration test (`cmd/export_test.go`)

Run full `export` command against `internal/extractor/testdata/` fixtures, verify `.qvs` files produced with expected content.

### Edge cases

| Scenario | Expected behaviour |
|---|---|
| No `///` marker | WARN, skip, continue |
| Invalid zlib data | ERR, skip, continue |
| Empty source dir | Exit 0, summary shows 0 |
| `--out` dir missing | Create dir tree, proceed |
| `--dry-run` | No files written, summary prefixed with "Dry run" |

## Dependencies

Provided by ckeletin-go skeleton:
- `github.com/spf13/cobra` — subcommand CLI framework
- `github.com/spf13/viper` — config/env var support
- `github.com/rs/zerolog` — structured logging
- `github.com/charmbracelet/bubbletea` — terminal UI
- `github.com/charmbracelet/lipgloss` — terminal styling

Stdlib only for core algorithm: `compress/zlib`, `io/fs`, `path/filepath`.
