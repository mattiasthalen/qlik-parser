---
layout: default
title: qlik-parser
---

<div class="hero">

# qlik-parser

Extract load scripts from QlikView (.qvw) and Qlik Sense (.qvf) files — fast, cross-platform, and scriptable.

<pre class="terminal"><code>qlik-parser extract --source ./qlik-apps --out ./scripts</code></pre>

<a href="https://github.com/mattiasthalen/qlik-parser/releases/latest" class="btn">Download</a>
<a href="https://github.com/mattiasthalen/qlik-parser" class="btn btn-outline">View on GitHub</a>

</div>

## Quick Start {#quick-start}

Get up and running in minutes. Download the binary for your platform, place it on your `PATH`, and run `qlik-parser extract` to pull all load scripts out of a directory of Qlik files.

## Installation {#install}

Download the pre-built binary for your platform from the [latest release](https://github.com/mattiasthalen/qlik-parser/releases/latest).

| Platform | Architecture | Download |
|----------|--------------|----------|
| Linux    | amd64        | qlik-parser_Linux_x86_64.tar.gz |
| Linux    | arm64        | qlik-parser_Linux_arm64.tar.gz |
| macOS    | amd64        | qlik-parser_Darwin_x86_64.tar.gz |
| macOS    | arm64        | qlik-parser_Darwin_arm64.tar.gz |
| Windows  | amd64        | qlik-parser_Windows_x86_64.zip |
| Windows  | arm64        | qlik-parser_Windows_arm64.zip |

**Linux / macOS**

```sh
tar -xzf qlik-parser_*.tar.gz
chmod +x qlik-parser
mv qlik-parser /usr/local/bin/
```

**Windows**

Extract the `.zip` file and add the folder containing `qlik-parser.exe` to your `PATH`.

## Usage {#usage}

### `extract`

Extract load scripts from all QlikView (.qvw) and Qlik Sense (.qvf) files in a directory.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--script` | | | Extract a single file by path |
| `--source` | `-s` | `./` | Source directory |
| `--out` | `-o` | `./out` | Output directory |
| `--dry-run` | | `false` | Preview without writing files |

Scripts are written to `<out>/<filename>.qvs`. Existing files are overwritten.

Preview what would be extracted without writing any files:

```sh
qlik-parser extract --source ./qlik-apps --dry-run
```

### `version`

Print the current version.

```sh
qlik-parser version
```

### Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--log-level` | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |
