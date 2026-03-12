# Documentation v0.1.0 — Design Spec

**Date:** 2026-03-12

---

## Overview

Three files to create for the v0.1.0 release:

1. `README.md` — end-user focused
2. `CONTRIBUTING.md` — contributor/developer focused
3. `LICENSE` — MIT license

---

## README.md

### Structure

1. **Header** — project name + badge row (CI, latest release, license)
2. **One-liner** — "Extract load scripts from QlikView (.qvw) and Qlik Sense (.qvf) files."
3. **Quick Start** — single concrete example command
4. **Installation** — download pre-built binary from GitHub Releases (Linux/macOS tar.gz, Windows zip, amd64 + arm64); note on making executable / adding to PATH
5. **Usage** — full flag reference for `extract` command + `version` command

### Flag Reference

**`extract` flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--script` | | true | Extract load scripts |
| `--source` | `-s` | current directory | Source directory to scan |
| `--out` | `-o` | alongside source files | Output directory |
| `--dry-run` | | false | Preview without writing files |

**Global flags (all commands):**

| Flag | Default | Description |
|------|---------|-------------|
| `--log-level` | `disabled` | Log level: debug, info, warn, error, disabled |

---

## CONTRIBUTING.md

### Structure

1. **Prerequisites** — Go version (from go.mod), devcontainer recommended
2. **Setup** — clone repo, open in devcontainer (or install Go manually)
3. **Make Commands** — two groups:

**Development:**

| Command | Description |
|---------|-------------|
| `make build` | Build the binary |
| `make test` | Run tests |
| `make cover` | Run tests with coverage report |
| `make lint` | Run linter |
| `make clean` | Remove built binary and coverage output |
| `make install-tools` | Install golangci-lint and svu (for fresh environments or version updates) |
| `make install-hooks` | Install git pre-commit hook |

**Maintainer / Release:**

| Command | Description |
|---------|-------------|
| `make next-version` | Preview the next semver tag |
| `make release` | Tag and push next semver version |
| `make release-patch` | Tag and push next patch version |
| `make release-minor` | Tag and push next minor version |
| `make release-major` | Tag and push next major version |

4. **Workflow** — create a git worktree for each feature (`git worktree add`), open a PR from the branch, conventional commits (feat:, fix:, chore:, etc.), CI must pass before merge. Direct commits to main are not allowed.

---

## LICENSE

MIT License, copyright holder: Mattias Thalen, year: 2026.

---

## Out of Scope

- Homebrew tap
- Architecture documentation
- `go install` instructions
- GitHub repo description/tags (manual step: description = "Extract load scripts from QlikView (.qvw) and Qlik Sense (.qvf) files", tags = qlikview, qlik-sense, qvw, qvf, cli, go, developer-tools, etl)
