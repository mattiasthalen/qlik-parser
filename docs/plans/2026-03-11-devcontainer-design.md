# Devcontainer Design

## Overview

Set up a devcontainer for developing a Go CLI, optimized for Claude Code (no VS Code extensions needed).

## Files

```
.devcontainer/
  devcontainer.json
  setup.sh
```

## devcontainer.json

- Base image: `mcr.microsoft.com/devcontainers/go:1.24`
- `postCreateCommand`: calls `.devcontainer/setup.sh`

## setup.sh

Installs:
- `golangci-lint` (latest via official install script)
- `make`

## Included via base image

- Go 1.24
- `gopls`
- `delve`
- `git`
