# GitHub Pages Site — Design Spec

**Date:** 2026-03-12
**Status:** Approved

---

## Overview

A single-page GitHub Pages site for `qlik-parser`: part landing page, part full reference docs. Dark, minimal, terminal aesthetic. Built with Jekyll and a hand-crafted custom theme. No build CI needed — GitHub Pages runs Jekyll natively.

---

## 1. Repository & Deployment Setup

- Jekyll site lives at `docs/site/` on `main`
- GitHub Pages configured to serve from `docs/site/` folder on `main` (set manually in repo Settings → Pages)
- No separate GitHub Actions workflow needed for deployment — GitHub Pages builds Jekyll automatically on push to `main`
- `.gitignore` additions: `_site/`, `.jekyll-cache/`

---

## 2. File Structure

```
docs/site/
  _config.yml            # Jekyll config: title, description, baseurl, no theme
  _layouts/
    default.html         # Custom layout: sticky nav, wraps {{ content }}, footer
  assets/
    css/
      style.css          # All custom CSS — dark palette, typography, code blocks, tables
  index.md               # Front matter + full page content in Markdown
```

No Jekyll theme is used. The `theme:` key is omitted from `_config.yml`. All styling is in `assets/css/style.css`.

---

## 3. Jekyll Configuration (`_config.yml`)

```yaml
title: qlik-parser
description: Extract load scripts from QlikView (.qvw) and Qlik Sense (.qvf) files.
baseurl: /qlik-parser
url: https://mattiasthalen.github.io
```

---

## 4. Visual Design

**Palette:**

| Token | Value | Usage |
|-------|-------|-------|
| Background | `#0f172a` | Page background |
| Surface | `#1e293b` | Code blocks, table rows |
| Border | `#334155` | Dividers, code block borders |
| Text primary | `#f1f5f9` | Headings, body |
| Text muted | `#94a3b8` | Subtext, nav links |
| Text subtle | `#64748b` | Labels, footer |
| Accent | `#0ea5e9` | Buttons, links, inline code highlight |
| Code text | `#7dd3fc` | Shell command text |

**Typography:** System sans-serif for body/nav; monospace for code blocks and the project name in the nav.

---

## 5. Page Layout

### Sticky Nav

Fixed to top on scroll. Contains:
- Left: `qlik-parser` in monospace
- Right: anchor links — `Quick Start`, `Install`, `Usage` — and a `GitHub ↗` external link

### Hero Section

- Project name (large, `#f1f5f9`)
- One-liner: "Extract load scripts from QlikView (.qvw) and Qlik Sense (.qvf) files."
- Terminal command block: `$ qlik-parser extract --source ./qlik-apps --out ./scripts`
- Two buttons:
  - **Download** — links to `https://github.com/mattiasthalen/qlik-parser/releases/latest` (no version label)
  - **View on GitHub** — links to `https://github.com/mattiasthalen/qlik-parser`

### Installation Section

- Platform table (Linux amd64/arm64, macOS amd64/arm64, Windows amd64/arm64 → `.tar.gz` or `.zip`)
- Shell commands for Linux/macOS (tar + chmod + mv)
- Windows instruction (extract zip, add to PATH)

### Usage Section

- `extract` command description
- Flag reference table: `--script`, `--source` / `-s`, `--out` / `-o`, `--dry-run`
- Output path behaviour note
- Dry-run example
- `version` command
- Global flags table: `--log-level`

### Footer

- "MIT License · Mattias Thalen"
- GitHub link

---

## 6. CI/CD

No changes to existing workflows. GitHub Pages deploys automatically when `docs/site/` changes on `main`. The Download button always points to `/releases/latest` — no version string in the button label or URL path.

---

## 7. Out of Scope

- Custom domain
- Search
- Dark/light mode toggle
- Versioned docs
- Changelog page
