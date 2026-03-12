# Rebrand to qlik-parser — Design Spec

**Date:** 2026-03-12
**Issue:** [#4](https://github.com/mattiasthalen/qlik-script-extractor/issues/4)
**Status:** Approved

## Summary

Rename the tool from `qlik-script-extractor` to `qlik-parser` to accommodate future artifact types beyond load scripts. Replace the `export` command with `extract` and add a `--script` flag as the first in a family of artifact-selection flags (`--variables`, `--charts` to follow in later issues).

No backwards compatibility is required — no release has been made yet.

## Scope

- Rename Go module path
- Rename binary (`Use` field in root command)
- Rename `export` command → `extract`
- Rename `cmd/export.go` → `cmd/extract.go`, `cmd/export_test.go` → `cmd/extract_test.go`
- Add `--script` boolean flag (default: `true`)
- Add validation: if all artifact flags are false, return exit code 1 with error message
- No changes to `internal/extractor` or `internal/ui`

## Command Interface

```
qlik-parser extract [flags]

Flags:
  --script              Extract load scripts (default: true)
  -s, --source string   Source directory to scan for .qvw files (default: current directory)
  -o, --out string      Export directory (default: alongside .qvw files)
  --dry-run             Show what would be extracted without writing files
  --log-level string    Log level: debug, info, warn, error, disabled (default: "disabled")
```

Note: `--script` has no short form; this is intentional.

**Bare invocation** (`qlik-parser extract`) works as today — `--script` defaults to true.

**No-op invocation** (`qlik-parser extract --script=false`) returns:
```
error: no artifact type selected
```
with exit code 1. Validation fires before any directory scanning occurs.

## Architecture

### Module rename

`go.mod` module path: `github.com/mattiasthalen/qlik-script-extractor` → `github.com/mattiasthalen/qlik-parser`

After updating `go.mod`, run `go mod tidy` to update `go.sum`. Then update all `.go` files containing the old import path. To find them all:

```
grep -r "qlik-script-extractor" --include="*.go" -l
```

### File changes

| File | Change |
|------|--------|
| `go.mod` | Module path renamed |
| `go.sum` | Updated via `go mod tidy` |
| `main.go` | Import path updated |
| `cmd/root.go` | `Use` field, descriptions, `AddCommand` call updated |
| `cmd/version.go` | Printed binary name updated |
| `cmd/root_test.go` | Hardcoded `"qlik-script-extractor"` assertion updated to `"qlik-parser"` |
| `cmd/export.go` | Renamed to `cmd/extract.go`; command name, flag, validation added |
| `cmd/export_test.go` | Renamed to `cmd/extract_test.go`; all `"export"` strings in `SetArgs` calls changed to `"extract"`, all `TestExportCmd_*` function names updated to `TestExtractCmd_*`, import path updated |
| `cmd/version_test.go` | Import path updated; binary-name string assertion updated to `"qlik-parser"` |
| `internal/extractor/*.go` / `*_test.go` | Import paths updated |
| `internal/ui/*.go` / `*_test.go` | Import paths updated |
| `internal/extractor/testdata/gen/*.go` | Import paths updated |
| `Makefile` | `BINARY := qlik-script-extractor` → `BINARY := qlik-parser` |
| `.gitignore` | Binary ignore entry updated to `qlik-parser` |
| `.goreleaser.yaml` | `binary:` field and `-X` ldflags path updated |

### root.go

- `Use`: `"qlik-script-extractor"` → `"qlik-parser"`
- `Short`/`Long` descriptions updated to reflect broader scope
- `AddCommand(newExportCmd())` → `AddCommand(newExtractCmd())`

### version.go

- Printed name: `"qlik-script-extractor %s\n"` → `"qlik-parser %s\n"`

### extract.go (was export.go)

- Command `Use`: `"export"` → `"extract"`
- New flag: `--script` bool, default `true`
- Validation at start of `RunE`: check that at least one artifact flag is true (written as a general guard, not `if !script`, to accommodate future `--variables` / `--charts` flags without revisiting this logic):

```go
if !script { // expand to: if !script && !variables && !charts as flags are added
    _, _ = fmt.Fprintf(cmd.ErrOrStderr(), "error: no artifact type selected\n")
    return ExitError(1)
}
```

- Validation fires before `Walk()` — no directory access occurs on invalid input
- Rest of logic unchanged — script extraction path runs when `script == true`

## Data Flow

```
qlik-parser extract [flags]
  └─ validate: at least one artifact flag is true
  └─ Walk(sourceDir) → qvwPaths
  └─ for each qvwPath:
       └─ if --script: ExtractScript → WriteScript
       └─ (future: if --variables: ExtractVariables → WriteVariables)
       └─ (future: if --charts: ExtractCharts → WriteCharts)
  └─ Summary
```

## Error Handling

| Condition | Behaviour |
|-----------|-----------|
| No artifact flag selected | `error: no artifact type selected`, exit 1, before any I/O |
| All other errors | Unchanged from current `export` behaviour |

## Testing

- `cmd/extract_test.go` — rename of `cmd/export_test.go`, all existing test cases ported
- `cmd/root_test.go` — update hardcoded `"qlik-script-extractor"` assertion to `"qlik-parser"`
- New test: `--script=false --source /nonexistent` → exit code 1, stderr contains `"no artifact type selected"` (not a filesystem error) — this proves the guard fires before any directory access
- All existing golden files and integration tests unchanged (internal package untouched)

## GitHub Repo Rename

After the code changes are committed and pushed, rename the GitHub repository:

1. GitHub → repository **Settings** → **General** → **Repository name** → change to `qlik-parser` → **Rename**
2. Update the remote in your local clone: `git remote set-url origin https://github.com/mattiasthalen/qlik-parser`

This step is done manually by the user after the implementation commit lands.

## Out of Scope

- `--variables` extraction (future issue)
- `--charts` extraction (future issue)
- Backwards compatibility shims for `export` command
