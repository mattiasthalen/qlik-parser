#!/usr/bin/env bash
set -euo pipefail

# Install make
sudo apt-get update -q && sudo apt-get install -y -q make && sudo rm -rf /var/lib/apt/lists/*

# Install golangci-lint (pinned version)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.5

echo "Setup complete."
