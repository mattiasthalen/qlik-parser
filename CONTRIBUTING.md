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
