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

**Setup:** Developers run `make install-hooks` once after cloning. This copies (not symlinks) `scripts/pre-commit` into `.git/hooks/pre-commit` to ensure Windows compatibility (symlinks require Developer Mode on Windows). The copy is idempotent — re-running overwrites.

**Platform note:** The pre-commit hook is a shell script and only runs on Unix (Linux/macOS) developers' machines. Windows developers using Git Bash or WSL are covered; native Windows `cmd.exe`/PowerShell users are not. This is acceptable because the CI gate (Section 2) provides the same protection for all contributors.

## 2. GitHub Actions CI

**File:** `.github/workflows/ci.yml`

**Triggers:**
- Pull requests (any branch → main)
- Tag pushes matching `v*`

**Matrix:** `ubuntu-latest` + `windows-latest`

**Steps per matrix leg:**
1. `actions/checkout`
2. `actions/setup-go` — version read from `go.mod` via `go-version-file: go.mod`
3. Go module cache (`actions/cache`) — cache key: `${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}`
4. `golangci-lint-action` — pin to a specific stable version (e.g. `v6`) recorded in the workflow file
5. `go test ./... -race -coverprofile=coverage.out`

Coverage is generated for log visibility but not enforced or uploaded.

**Rationale for two OSes:** The tool targets Windows as its primary platform. Running CI on both Linux and Windows catches OS-specific path or file-handling issues early. macOS is covered by the cross-compiled release artifacts.

**Note on `-race` on Windows:** The Go race detector is supported on Windows/amd64. If it causes flakiness in future, it can be conditionally disabled for the Windows leg.

## 3. GitHub Actions Release

**File:** `.github/workflows/release.yml`

**Triggers:**
- Tag push matching `v*`
- Manual workflow dispatch

**CI gate strategy:** The release workflow does NOT use `needs:` to depend on `ci.yml` (cross-workflow `needs:` is not supported in GitHub Actions). Instead, the release is protected by two layers:
1. The repo enforces PRs — no code reaches `main` without passing CI.
2. The release workflow includes a `verify` job that duplicates the CI steps (lint + test on `ubuntu-latest` only) before the `release` job runs. The `release` job uses `needs: verify`.

**Manual dispatch pre-condition:** Before triggering manually, the caller must ensure the target commit has already been tagged with a valid semver tag (e.g. `v1.2.3`). The workflow does not create tags itself. GoReleaser will fail if the HEAD commit has no matching tag.

**Jobs:**

### `verify` job
- Runs on `ubuntu-latest`
- Checkout with `fetch-depth: 0`
- Setup Go + cache
- `golangci-lint-action`
- `go test ./... -race`

### `release` job (needs: verify)
- Runs on `ubuntu-latest`
- `actions/checkout` with `fetch-depth: 0`
- `actions/setup-go`
- `goreleaser/goreleaser-action` — pin to a specific version (e.g. `v6`) recorded in the workflow file

**Secrets:** Only `GITHUB_TOKEN` (built-in GitHub secret, no manual setup required).

## 4. GoReleaser Configuration

**File:** `.goreleaser.yaml`

**Schema version:** File must begin with `version: 2` (required by GoReleaser v2+).

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

**`make install-tools`** installs all required local tools using these exact commands:

```makefile
install-tools:
	go install github.com/caarlos0/svu@latest
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin
```

`svu` is installed via `go install` (idiomatic for Go tools).
`golangci-lint` is installed via the official install script to a versioned binary in `$GOPATH/bin` (the `go install` path is officially unsupported for golangci-lint).

**`make install-hooks`** copies the pre-commit script:

```makefile
install-hooks:
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
```

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
