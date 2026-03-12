# Documentation v0.1.0 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create README.md, CONTRIBUTING.md, and LICENSE for the v0.1.0 release of qlik-parser.

**Architecture:** Three static documentation files at the repo root. No code changes. Each file is self-contained and written directly.

**Tech Stack:** Markdown, MIT License text, GitHub badge URLs.

**Worktree:** `.worktrees/docs/v0.1.0` (branch `docs/v0.1.0`)

> All `git` and `gh` commands below must be run from inside the worktree:
> ```bash
> cd /workspaces/qlik-parser/.worktrees/docs/v0.1.0
> ```

---

## Chunk 1: LICENSE

### Task 1: Create LICENSE

**Files:**
- Create: `/workspaces/qlik-parser/.worktrees/docs/v0.1.0/LICENSE`

- [ ] **Step 1: Create the LICENSE file**

Create `/workspaces/qlik-parser/.worktrees/docs/v0.1.0/LICENSE` with this exact content:

```
MIT License

Copyright (c) 2026 Mattias Thalen

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 2: Commit and push**

```bash
cd /workspaces/qlik-parser/.worktrees/docs/v0.1.0
git add LICENSE
git commit -m "docs: add MIT license"
git push -u origin docs/v0.1.0
```

---

## Chunk 2: README.md

### Task 2: Create README.md

**Files:**
- Create: `/workspaces/qlik-parser/.worktrees/docs/v0.1.0/README.md`

- [ ] **Step 1: Create README.md**

Create `/workspaces/qlik-parser/.worktrees/docs/v0.1.0/README.md` with this exact content:

```markdown
# qlik-parser

[![CI](https://github.com/mattiasthalen/qlik-parser/actions/workflows/ci.yml/badge.svg)](https://github.com/mattiasthalen/qlik-parser/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/mattiasthalen/qlik-parser)](https://github.com/mattiasthalen/qlik-parser/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Extract load scripts from QlikView (.qvw) and Qlik Sense (.qvf) files.

## Quick Start

```sh
qlik-parser extract --source ./qlik-apps --out ./scripts
```

This scans `./qlik-apps` recursively for `.qvw` and `.qvf` files and writes the extracted load scripts to `./scripts`, mirroring the source folder structure.

## Installation

Download the binary for your platform from the [Releases page](https://github.com/mattiasthalen/qlik-parser/releases/latest).

| Platform | Archive |
|----------|---------|
| Linux (amd64 / arm64) | `.tar.gz` |
| macOS (amd64 / arm64) | `.tar.gz` |
| Windows (amd64 / arm64) | `.zip` |

**Linux / macOS:**

```sh
tar -xzf qlik-parser_<version>_<os>_<arch>.tar.gz
chmod +x qlik-parser
mv qlik-parser /usr/local/bin/   # or any directory on your PATH
```

**Windows:**

Extract the `.zip` and move `qlik-parser.exe` to a directory on your `PATH`.

## Usage

### `extract`

Recursively scans `--source` for `.qvw` and `.qvf` files and extracts embedded load scripts.

```
qlik-parser extract [flags]
```

**Flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--script` | | `true` | Extract load scripts |
| `--source` | `-s` | current directory | Source directory to scan |
| `--out` | `-o` | alongside source files | Output directory |
| `--dry-run` | | `false` | Preview without writing files |

**Output path behaviour:**

- `--out` specified: mirrors source folder structure under the output directory
- `--out` omitted: writes `.qvs` files alongside the source files

**Example — dry run:**

```sh
qlik-parser extract --source ./qlik-apps --dry-run
```

### `version`

```sh
qlik-parser version
```

Prints the current version, e.g. `qlik-parser v0.1.0`.

### Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--log-level` | `disabled` | Log level: `debug`, `info`, `warn`, `error`, `disabled` |
```

- [ ] **Step 2: Commit and push**

```bash
cd /workspaces/qlik-parser/.worktrees/docs/v0.1.0
git add README.md
git commit -m "docs: add README"
git push
```

---

## Chunk 3: CONTRIBUTING.md

### Task 3: Create CONTRIBUTING.md

**Files:**
- Create: `/workspaces/qlik-parser/.worktrees/docs/v0.1.0/CONTRIBUTING.md`

- [ ] **Step 1: Create CONTRIBUTING.md**

Create `/workspaces/qlik-parser/.worktrees/docs/v0.1.0/CONTRIBUTING.md` with this exact content:

```markdown
# Contributing

## Prerequisites

- Go 1.25.0 (see `go.mod`)
- A devcontainer-capable editor (recommended) — Go, golangci-lint, and all tools are pre-installed

## Setup

```sh
git clone https://github.com/mattiasthalen/qlik-parser.git
cd qlik-parser
```

Open in a devcontainer (VS Code: **Reopen in Container**), or install Go 1.25.0 manually.

Run `make install-hooks` once to install the pre-commit hook.

## Make Commands

### Development

| Command | Description |
|---------|-------------|
| `make build` | Build the binary |
| `make test` | Run tests |
| `make cover` | Run tests with coverage report |
| `make lint` | Run linter |
| `make clean` | Remove built binary and coverage output |
| `make install-tools` | Install golangci-lint and svu (for fresh environments or version updates) |
| `make install-hooks` | Install git pre-commit hook |

### Maintainer / Release

| Command | Description |
|---------|-------------|
| `make next-version` | Preview the next semver tag |
| `make release` | Tag and push next semver version |
| `make release-patch` | Tag and push next patch version |
| `make release-minor` | Tag and push next minor version |
| `make release-major` | Tag and push next major version |

## Workflow

1. Create a worktree for your feature branch:
   ```sh
   git worktree add .worktrees/<feature> -b <feature>
   cd .worktrees/<feature>
   ```
2. Make your changes. Commits must follow [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `chore:`, `docs:`, etc.
3. Open a pull request against `main`. CI must pass before merge.
4. Direct commits to `main` are not allowed.
```

- [ ] **Step 2: Commit and push**

```bash
cd /workspaces/qlik-parser/.worktrees/docs/v0.1.0
git add CONTRIBUTING.md
git commit -m "docs: add CONTRIBUTING"
git push
```

---

## Chunk 4: Open PR

- [ ] **Step 1: Open PR**

```bash
cd /workspaces/qlik-parser/.worktrees/docs/v0.1.0
gh pr create \
  --title "docs: add README, CONTRIBUTING, and LICENSE for v0.1.0" \
  --body "$(cat <<'EOF'
Adds the three documentation files required for the v0.1.0 release.

## Changes
- `README.md` — end-user quick start, installation, and full flag reference
- `CONTRIBUTING.md` — dev setup, make commands, workflow conventions
- `LICENSE` — MIT, copyright Mattias Thalen 2026

## Manual follow-up
After merge, update the GitHub repo description and topics:
- **Description:** Extract load scripts from QlikView (.qvw) and Qlik Sense (.qvf) files
- **Topics:** qlikview, qlik-sense, qvw, qvf, cli, go, developer-tools, etl
EOF
)"
```
