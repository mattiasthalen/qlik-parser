# CI/CD Design — qlik-script-extractor

**Date:** 2026-03-12
**Status:** Approved

## Overview

Set up a complete CI/CD pipeline for the `qlik-script-extractor` CLI:

- Local pre-commit hook for fast feedback (lint + test before every commit)
- GitHub Actions CI gate on PRs and tag pushes
- GitHub Actions Release pipeline via GoReleaser publishing to GitHub Releases
- Semantic version bumping via `svu` + Makefile targets

## 1. Local Pre-commit Hook

**Purpose:** Block commits that break lint or tests, giving developers immediate feedback before code leaves the machine.

**Files:**
- `scripts/pre-commit` — executable shell script committed to the repo
- `Makefile` — new `install-hooks` target

**Hook behaviour:**
1. Runs `golangci-lint run ./...`
2. Runs `go test ./...`
3. Exits non-zero (blocking the commit) if either step fails

**Setup:** Developers run `make install-hooks` once after cloning. This symlinks `scripts/pre-commit` into `.git/hooks/pre-commit`.

## 2. GitHub Actions CI

**File:** `.github/workflows/ci.yml`

**Triggers:**
- Pull requests (any branch → main)
- Tag pushes matching `v*`

**Matrix:** `ubuntu-latest` + `windows-latest`

**Steps per matrix leg:**
1. `actions/checkout`
2. `actions/setup-go` — version read from `go.mod`
3. Go module cache (`actions/cache`)
4. `golangci-lint-action`
5. `go test ./... -race -coverprofile=coverage.out`

Coverage is generated for log visibility but not enforced or uploaded.

**Rationale for two OSes:** The tool targets Windows as its primary platform. Running CI on both Linux and Windows catches OS-specific path or file-handling issues early. macOS is covered by the cross-compiled release artifacts.

## 3. GitHub Actions Release

**File:** `.github/workflows/release.yml`

**Triggers:**
- Tag push matching `v*`
- Manual workflow dispatch (with optional version notes input)

**Dependency:** The `release` job requires the `ci` job to pass first (via `needs: ci` or a separate reusable call).

**Steps:**
1. `actions/checkout` with `fetch-depth: 0` (full history required for `git describe`)
2. `actions/setup-go`
3. `goreleaser/goreleaser-action` with `GITHUB_TOKEN` secret

**Secrets:** Only `GITHUB_TOKEN` (built-in GitHub secret, no manual setup required).

## 4. GoReleaser Configuration

**File:** `.goreleaser.yaml`

**Build targets (6 binaries):**

| OS      | amd64 | arm64 |
|---------|-------|-------|
| linux   | ✓     | ✓     |
| darwin  | ✓     | ✓     |
| windows | ✓     | ✓     |

**Version injection:**
```
-ldflags "-X github.com/mattiasthalen/qlik-script-extractor/cmd.Version={{ .Version }}"
```
GoReleaser populates `{{ .Version }}` from the git tag (e.g. `v1.2.3`).

**Archives:**
- `.tar.gz` for linux and darwin
- `.zip` for windows

**Extras:** `checksums.txt` (SHA256), auto-generated changelog from conventional commits grouped by `feat:` and `fix:`.

## 5. Dynamic Version Bumping

**Tool:** [`svu`](https://github.com/caarlos0/svu) — reads git tags and conventional commit messages to compute the next semantic version.

**Version inference rules (from conventional commits):**
- `BREAKING CHANGE` in footer or `!` after type → major bump
- `feat:` → minor bump
- `fix:`, `chore:`, etc. → patch bump

**Makefile targets:**

| Target | Behaviour |
|--------|-----------|
| `make next-version` | Print the next version (dry run) |
| `make release` | Tag with `svu next` + push tag |
| `make release-patch` | Tag with `svu patch` + push tag |
| `make release-minor` | Tag with `svu minor` + push tag |
| `make release-major` | Tag with `svu major` + push tag |

**Typical workflow:**
```
make next-version   # preview: v1.3.0
make release        # tags v1.3.0, pushes → triggers GoReleaser
```

## 6. Developer Tooling

**`make install-tools`** installs all required local tools:
- `golangci-lint`
- `svu`

**`make install-hooks`** wires up the pre-commit hook.

New contributor setup:
```
make install-tools
make install-hooks
```

## File Inventory

```
.github/
  workflows/
    ci.yml
    release.yml
.goreleaser.yaml
scripts/
  pre-commit
Makefile          (updated: install-tools, install-hooks, release targets)
```
