# Devcontainer Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a devcontainer for Go CLI development using a pre-built Microsoft Go image with golangci-lint and make added via a setup script.

**Architecture:** A single `devcontainer.json` pointing at the official Go 1.24 devcontainer image, with a `setup.sh` script run as `postCreateCommand` to install additional tools.

**Tech Stack:** Docker, devcontainer spec, Go 1.24, golangci-lint, make

---

### Task 1: Create `.devcontainer/devcontainer.json`

**Files:**
- Create: `.devcontainer/devcontainer.json`

**Step 1: Create the file**

```json
{
  "name": "Go CLI",
  "image": "mcr.microsoft.com/devcontainers/go:1.24",
  "postCreateCommand": "bash .devcontainer/setup.sh",
  "remoteEnv": {
    "GOPATH": "/go",
    "PATH": "${PATH}:/go/bin"
  }
}
```

**Step 2: Verify the file is valid JSON**

Run: `cat .devcontainer/devcontainer.json | python3 -m json.tool`
Expected: JSON printed without errors

**Step 3: Commit**

```bash
git add .devcontainer/devcontainer.json
git commit -m "feat: add devcontainer.json with Go 1.24 image"
```

---

### Task 2: Create `.devcontainer/setup.sh`

**Files:**
- Create: `.devcontainer/setup.sh`

**Step 1: Create the file**

```bash
#!/usr/bin/env bash
set -euo pipefail

# Install make
sudo apt-get update -q && sudo apt-get install -y -q make

# Install golangci-lint (latest)
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(go env GOPATH)/bin"

echo "Setup complete."
```

**Step 2: Make it executable**

Run: `chmod +x .devcontainer/setup.sh`

**Step 3: Verify shebang and syntax**

Run: `bash -n .devcontainer/setup.sh`
Expected: No output (no syntax errors)

**Step 4: Commit**

```bash
git add .devcontainer/setup.sh
git commit -m "feat: add devcontainer setup script"
```
