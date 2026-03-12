# Design: Fix Docs After Artifact Extraction PR

**Date:** 2026-03-12
**Status:** Approved

## Overview

The artifact extraction PR (#12) introduced breaking changes to the `extract` command. The existing documentation in `README.md` and `docs/index.md` is now stale. This update corrects the broken content with minimal scope — no restructuring, no new sections.

`CONTRIBUTING.md` is unaffected.

---

## 1. Scope

| File | What's broken | Fix |
|------|--------------|-----|
| `README.md` | (1) Tagline (line 7): "Extract load scripts from QlikView..." — implies scripts only. (2) Flag table: shows `--script` default `true`, no artifact flags. (3) Quick Start paragraph (line 15): says "writes the extracted load scripts to `./scripts`" — implies scripts only. (4) `extract` section intro (line 43): "extracts embedded load scripts" — implies scripts only. (5) "Output path behaviour" bullets: second bullet says "writes `.qvs` files alongside the source files". | Replace tagline, replace flag table, update Quick Start paragraph, update extract section intro, replace output bullets, add selection behavior note |
| `docs/index.md` | (1) Hero tagline (line 10): "Extract load scripts from QlikView..." — implies scripts only. (2) Flag table: `--script` description says "Extract a single file by path" (wrong text, leftover), no artifact flags, `--out` default shows `./out` instead of alongside. (3) Output description says `<out>/<filename>.qvs`. (4) Quick Start paragraph says "pull all load scripts out of a directory of Qlik files" — implies scripts only. (5) `extract` command description under Usage says "Extract load scripts from all..." — implies scripts only. (6) Hero `<div>` doesn't render markdown — Jekyll doesn't process markdown inside HTML blocks by default. | Replace hero tagline, replace entire flag table, fix output description, update Quick Start paragraph, update extract command description, add `markdown="1"` to hero div |

---

## 2. Flag Table (both files)

Replace the entire existing flag table in both files with:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--script` | | `false` | Extract load scripts |
| `--measures` | | `false` | Extract master measures (QVF only) |
| `--dimensions` | | `false` | Extract master dimensions (QVF only) |
| `--variables` | | `false` | Extract variables (QVF only) |
| `--source` | `-s` | current directory | Source directory to scan |
| `--out` | `-o` | alongside source files | Output directory |
| `--dry-run` | | `false` | Preview without writing files |

Add a short note after the table:

> No artifact flags passed → all artifacts extracted. Explicit flags → only those artifact types.

---

## 3. Output Description (both files)

Each source file gets its own output folder named `<filename>.<ext>/`. This applies to both `.qvf` and `.qvw` files.

Examples:
- `sales.qvf` → `sales.qvf/script.qvs`, `sales.qvf/measures.json`, `sales.qvf/dimensions.json`, `sales.qvf/variables.json`
- `report.qvw` → `report.qvw/script.qvs`

### `README.md` replacements

**Tagline** (line 7) — replace:

Old: `Extract load scripts from QlikView (.qvw) and Qlik Sense (.qvf) files.`

New: `Extract artifacts from QlikView (.qvw) and Qlik Sense (.qvf) files.`

**`extract` section intro** (line 43, under `### extract`) — replace:

Old: `Recursively scans \`--source\` for \`.qvw\` and \`.qvf\` files and extracts embedded load scripts.`

New: `Recursively scans \`--source\` for \`.qvw\` and \`.qvf\` files and extracts embedded artifacts.`

**Quick Start paragraph** — replace the sentence:

Old: `This scans \`./qlik-apps\` recursively for \`.qvw\` and \`.qvf\` files and writes the extracted load scripts to \`./scripts\`, mirroring the source folder structure.`

New: `This scans \`./qlik-apps\` recursively for \`.qvw\` and \`.qvf\` files and writes extracted artifacts to \`./scripts\`, creating a folder per source file (e.g. \`./scripts/sales.qvf/script.qvs\`).`

**"Output path behaviour" bullets** — replace both bullets:

Old:
```
- `--out` specified: mirrors source folder structure under the output directory
- `--out` omitted: writes `.qvs` files alongside the source files
```

New:
```
- `--out` specified: mirrors source folder structure under the output directory, one folder per source file
- `--out` omitted: creates a folder per source file alongside the source
```

### `docs/index.md` replacements

**Hero tagline** (line 10, inside the hero div) — replace:

Old: `Extract load scripts from QlikView (.qvw) and Qlik Sense (.qvf) files.`

New: `Extract artifacts from QlikView (.qvw) and Qlik Sense (.qvf) files.`

**Quick Start paragraph** — replace the sentence:

Old: `run \`qlik-parser extract\` to pull all load scripts out of a directory of Qlik files.`

New: `run \`qlik-parser extract\` to extract all artifacts out of a directory of Qlik files.`

**`extract` command description** (under Usage) — replace the sentence:

Old: `Extract load scripts from all QlikView (.qvw) and Qlik Sense (.qvf) files in a directory.`

New: `Extract artifacts from all QlikView (.qvw) and Qlik Sense (.qvf) files in a directory.`

**Output line** — replace:

Old: `Scripts are written to \`<out>/<filename>.qvs\`. Existing files are overwritten.`

New: `Each source file produces a folder named \`<filename>.<ext>/\` containing the extracted artifacts. Existing files are overwritten.`

---

## 4. Hero Div Fix (`docs/index.md` only)

Change `<div class="hero">` to `<div class="hero" markdown="1">` so Jekyll/kramdown processes the markdown inside the block and renders `# qlik-parser` as an `<h1>` instead of literal text.

---

## 5. Out of Scope

- Restructuring or reorganizing content
- Adding new usage examples
- Changelog or release notes
- `CONTRIBUTING.md`
- `--log-level` default discrepancy between `README.md` (`disabled`) and `docs/index.md` (`info`) — left for a separate cleanup
