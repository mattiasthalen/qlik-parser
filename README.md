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
