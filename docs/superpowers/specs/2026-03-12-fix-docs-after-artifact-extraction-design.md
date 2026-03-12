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
| `README.md` | Flag table shows `--script` default `true`; no artifact flags; output described as flat `.qvs` files | Update flag table, add selection behavior note, fix output description |
| `docs/index.md` | Same flag table issues; hero `<div>` doesn't render markdown (Jekyll doesn't process markdown inside HTML blocks by default) | Same flag/output fixes + add `markdown="1"` to hero div |

---

## 2. Flag Table (both files)

Replace old table with:

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

Old: "writes `.qvs` files alongside the source files" / "scripts are written to `<out>/<filename>.qvs`"

New: each source file gets its own output folder named `<filename>.<ext>/`. Example: `sales.qvf` → `sales.qvf/script.qvs`, `sales.qvf/measures.json`, etc.

---

## 4. Hero Div Fix (`docs/index.md` only)

Change `<div class="hero">` to `<div class="hero" markdown="1">` so Jekyll/kramdown processes the markdown inside the block and renders `# qlik-parser` as an `<h1>` instead of literal text.

---

## 5. Out of Scope

- Restructuring or reorganizing content
- Adding new usage examples
- Changelog or release notes
- `CONTRIBUTING.md`
