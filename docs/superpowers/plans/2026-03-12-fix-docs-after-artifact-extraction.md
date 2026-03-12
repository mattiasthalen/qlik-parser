# Fix Docs After Artifact Extraction Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Correct stale documentation in `README.md` and `docs/index.md` after the artifact extraction PR (#12) changed the `extract` command's flags and output behavior.

**Architecture:** Pure documentation edits — no code changes, no tests. Two files are modified independently; each task targets one file. All changes are specified verbatim in the spec.

**Tech Stack:** Markdown, Jekyll/kramdown (for `docs/index.md` `markdown="1"` attribute)

---

## Chunk 1: README.md

### Task 1: Fix README.md

**Files:**
- Modify: `README.md`

There are six targeted edits. Apply them in order — each is a precise string replacement.

- [ ] **Step 1: Replace the tagline (line 7)**

Old:
```
Extract load scripts from QlikView (.qvw) and Qlik Sense (.qvf) files.
```

New:
```
Extract artifacts from QlikView (.qvw) and Qlik Sense (.qvf) files.
```

- [ ] **Step 2: Replace the Quick Start paragraph (line 15)**

Old:
```
This scans `./qlik-apps` recursively for `.qvw` and `.qvf` files and writes the extracted load scripts to `./scripts`, mirroring the source folder structure.
```

New:
```
This scans `./qlik-apps` recursively for `.qvw` and `.qvf` files and writes extracted artifacts to `./scripts`, creating a folder per source file (e.g. `./scripts/sales.qvf/script.qvs`).
```

- [ ] **Step 3: Replace the `extract` section intro (line 43)**

Old:
```
Recursively scans `--source` for `.qvw` and `.qvf` files and extracts embedded load scripts.
```

New:
```
Recursively scans `--source` for `.qvw` and `.qvf` files and extracts embedded artifacts.
```

- [ ] **Step 4: Replace the flag table**

Old:
```
| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--script` | | `true` | Extract load scripts |
| `--source` | `-s` | current directory | Source directory to scan |
| `--out` | `-o` | alongside source files | Output directory |
| `--dry-run` | | `false` | Preview without writing files |
```

New:
```
| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--script` | | `false` | Extract load scripts |
| `--measures` | | `false` | Extract master measures (QVF only) |
| `--dimensions` | | `false` | Extract master dimensions (QVF only) |
| `--variables` | | `false` | Extract variables (QVF only) |
| `--source` | `-s` | current directory | Source directory to scan |
| `--out` | `-o` | alongside source files | Output directory |
| `--dry-run` | | `false` | Preview without writing files |
```

- [ ] **Step 5: Add selection behavior note after the flag table**

After the flag table, add a blank line then:
```
> No artifact flags passed → all artifacts extracted. Explicit flags → only those artifact types.
```

- [ ] **Step 6: Replace the "Output path behaviour" bullets**

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

- [ ] **Step 7: Verify the final README.md looks correct**

Open `README.md` and confirm:
- Tagline says "Extract artifacts from..."
- Quick Start paragraph mentions "creating a folder per source file"
- `extract` intro says "extracts embedded artifacts"
- Flag table has 7 rows including `--measures`, `--dimensions`, `--variables` with default `false`
- Selection behavior note is present
- Output bullets mention "one folder per source file" / "creates a folder per source file"

- [ ] **Step 8: Commit**

```bash
git add README.md
git commit -m "docs: update README.md for artifact extraction changes"
```

---

## Chunk 2: docs/index.md

### Task 2: Fix docs/index.md

**Files:**
- Modify: `docs/index.md`

There are six targeted edits. Apply them in order.

- [ ] **Step 1: Add `markdown="1"` to the hero div (line 6)**

Old:
```
<div class="hero">
```

New:
```
<div class="hero" markdown="1">
```

- [ ] **Step 2: Replace the hero tagline (line 10)**

Old:
```
Extract load scripts from QlikView (.qvw) and Qlik Sense (.qvf) files.
```

New:
```
Extract artifacts from QlikView (.qvw) and Qlik Sense (.qvf) files.
```

- [ ] **Step 3: Replace the Quick Start paragraph (line 21)**

Old:
```
Get up and running in minutes. Download the binary for your platform, place it on your `PATH`, and run `qlik-parser extract` to pull all load scripts out of a directory of Qlik files.
```

New:
```
Get up and running in minutes. Download the binary for your platform, place it on your `PATH`, and run `qlik-parser extract` to extract all artifacts out of a directory of Qlik files.
```

- [ ] **Step 4: Replace the `extract` command description (line 52)**

Old:
```
Extract load scripts from all QlikView (.qvw) and Qlik Sense (.qvf) files in a directory.
```

New:
```
Extract artifacts from all QlikView (.qvw) and Qlik Sense (.qvf) files in a directory.
```

- [ ] **Step 5: Replace the entire flag table and add selection behavior note**

Old:
```
| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--script` | | | Extract a single file by path |
| `--source` | `-s` | `./` | Source directory |
| `--out` | `-o` | `./out` | Output directory |
| `--dry-run` | | `false` | Preview without writing files |
```

New (replace the entire old table with this, including the note on the line after):
```
| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--script` | | `false` | Extract load scripts |
| `--measures` | | `false` | Extract master measures (QVF only) |
| `--dimensions` | | `false` | Extract master dimensions (QVF only) |
| `--variables` | | `false` | Extract variables (QVF only) |
| `--source` | `-s` | current directory | Source directory to scan |
| `--out` | `-o` | alongside source files | Output directory |
| `--dry-run` | | `false` | Preview without writing files |

> No artifact flags passed → all artifacts extracted. Explicit flags → only those artifact types.
```

- [ ] **Step 6: Replace the output description line (line 61)**

Old:
```
Scripts are written to `<out>/<filename>.qvs`. Existing files are overwritten.
```

New:
```
Each source file produces a folder named `<filename>.<ext>/` containing the extracted artifacts. Existing files are overwritten.
```

- [ ] **Step 7: Verify the final docs/index.md looks correct**

Open `docs/index.md` and confirm:
- Hero div tag has `markdown="1"`
- Hero tagline says "Extract artifacts from..."
- Quick Start paragraph says "extract all artifacts out of a directory..."
- `extract` command description says "Extract artifacts from all..."
- Flag table has 7 rows with correct defaults (all artifact flags `false`, `--out` default `alongside source files`)
- Selection behavior note is present
- Output description says "produces a folder named `<filename>.<ext>/`"

- [ ] **Step 8: Commit**

```bash
git add docs/index.md
git commit -m "docs: update docs/index.md for artifact extraction changes"
```

---

## Chunk 3: Push

- [ ] **Step 1: Push the branch**

```bash
git push --set-upstream origin docs/fix-docs-after-artifact-extraction
```

Expected: both commits pushed to origin on the `docs/fix-docs-after-artifact-extraction` branch.
