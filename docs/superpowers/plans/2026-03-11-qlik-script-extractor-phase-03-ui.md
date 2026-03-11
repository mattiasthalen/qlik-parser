# QlikView Script Extractor — Phase 03: Terminal UI Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `internal/ui/output.go` — the bubbletea/lipgloss terminal UI with spinner, per-file status lines, summary, TTY detection, and dry-run mode.

**Architecture:** A single `output.go` file exposing a `Printer` type. `Printer` is constructed once, accepts `FileResult` events sequentially, and prints a final summary. bubbletea is used for the spinner in TTY mode; in non-TTY mode output is plain text to stdout. No extraction logic lives here.

**Tech Stack:** Go 1.24, `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss`, stdlib `os`, `fmt`, `strings`.

**Spec:** `docs/superpowers/specs/2026-03-11-qlik-script-extractor-design.md` — "Terminal UI" section.

**Prerequisites:** Phase 01 complete (module, deps). Phase 02 complete (extractor types — `NoScriptError`).

**Parallelism note:** This phase is fully independent of Phase 02's internal logic. Tasks 1–3 can be built in a separate worktree from Phase 02, then merged.

---

## Chunk 1: UI Types and Plain-Text Output

### Task 1: Write UI types and non-TTY printer test

**Files:**
- Create: `internal/ui/output_test.go`

- [ ] **Step 1: Write tests for plain-text (non-TTY) output**

```go
package ui_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/your-org/qlik-script-extractor/internal/ui"
)

func newTestPrinter(buf *bytes.Buffer) *ui.Printer {
	// Force non-TTY mode by passing a plain bytes.Buffer as writer
	return ui.NewPrinter(buf, false, false)
}

func TestPrinter_SuccessLine(t *testing.T) {
	buf := &bytes.Buffer{}
	p := newTestPrinter(buf)
	p.FileResult(ui.Result{
		Status:    ui.StatusOK,
		QVWPath:   "sales.qvw",
		QVSPath:   "sales.qvs",
		CharCount: 4821,
	})
	out := buf.String()
	if !strings.Contains(out, "sales.qvw") {
		t.Errorf("expected qvw name in output, got: %q", out)
	}
	if !strings.Contains(out, "sales.qvs") {
		t.Errorf("expected qvs name in output, got: %q", out)
	}
	if !strings.Contains(out, "4,821") {
		t.Errorf("expected char count in output, got: %q", out)
	}
}

func TestPrinter_WarnLine(t *testing.T) {
	buf := &bytes.Buffer{}
	p := newTestPrinter(buf)
	p.FileResult(ui.Result{
		Status:  ui.StatusWarn,
		QVWPath: "empty.qvw",
		Message: "no script found",
	})
	out := buf.String()
	if !strings.Contains(out, "empty.qvw") {
		t.Errorf("expected filename in warn output, got: %q", out)
	}
	if !strings.Contains(out, "no script found") {
		t.Errorf("expected message in warn output, got: %q", out)
	}
}

func TestPrinter_ErrLine(t *testing.T) {
	buf := &bytes.Buffer{}
	p := newTestPrinter(buf)
	p.FileResult(ui.Result{
		Status:  ui.StatusErr,
		QVWPath: "corrupt.qvw",
		Message: "zlib: invalid header",
	})
	out := buf.String()
	if !strings.Contains(out, "corrupt.qvw") {
		t.Errorf("expected filename in err output, got: %q", out)
	}
	if !strings.Contains(out, "zlib: invalid header") {
		t.Errorf("expected error message, got: %q", out)
	}
}

func TestPrinter_DryRunSuffix(t *testing.T) {
	buf := &bytes.Buffer{}
	p := ui.NewPrinter(buf, false, true) // dryRun=true
	p.FileResult(ui.Result{
		Status:    ui.StatusOK,
		QVWPath:   "sales.qvw",
		QVSPath:   "sales.qvs",
		CharCount: 100,
	})
	out := buf.String()
	if !strings.Contains(out, "[dry run]") {
		t.Errorf("expected '[dry run]' suffix in dry-run output, got: %q", out)
	}
}

func TestPrinter_Summary_Normal(t *testing.T) {
	buf := &bytes.Buffer{}
	p := newTestPrinter(buf)
	p.FileResult(ui.Result{Status: ui.StatusOK, QVWPath: "a.qvw", QVSPath: "a.qvs"})
	p.FileResult(ui.Result{Status: ui.StatusOK, QVWPath: "b.qvw", QVSPath: "b.qvs"})
	p.FileResult(ui.Result{Status: ui.StatusWarn, QVWPath: "c.qvw", Message: "no script"})
	p.FileResult(ui.Result{Status: ui.StatusErr, QVWPath: "d.qvw", Message: "corrupt"})
	p.Summary()
	out := buf.String()
	if !strings.Contains(out, "Extracted 2 scripts") {
		t.Errorf("expected 'Extracted 2 scripts' in summary, got: %q", out)
	}
}

func TestPrinter_Summary_DryRun(t *testing.T) {
	buf := &bytes.Buffer{}
	p := ui.NewPrinter(buf, false, true)
	p.FileResult(ui.Result{Status: ui.StatusOK, QVWPath: "a.qvw", QVSPath: "a.qvs"})
	p.FileResult(ui.Result{Status: ui.StatusWarn, QVWPath: "b.qvw", Message: "no script"})
	p.Summary()
	out := buf.String()
	// Spec: "Dry run — 10 files would be extracted  ✓ 10  ⚠ 1  ✗ 1"
	// With 1 OK + 1 WARN, total = 2
	if !strings.Contains(out, "Dry run — 2 files would be extracted") {
		t.Errorf("expected 'Dry run — 2 files would be extracted' in summary, got: %q", out)
	}
}

func TestPrinter_Summary_ZeroFiles(t *testing.T) {
	buf := &bytes.Buffer{}
	p := newTestPrinter(buf)
	p.Summary()
	out := buf.String()
	if !strings.Contains(out, "Extracted 0 scripts") {
		t.Errorf("expected 'Extracted 0 scripts' in zero-file summary, got: %q", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/... -v`
Expected: FAIL — `ui` package does not exist.

---

### Task 2: Implement output.go

**Files:**
- Create: `internal/ui/output.go`

- [ ] **Step 1: Write output.go**

```go
package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Status represents the outcome of processing a single .qvw file.
type Status int

const (
	StatusOK   Status = iota // Script successfully extracted
	StatusWarn               // Non-fatal warning (e.g. no script found)
	StatusErr                // Fatal per-file error (e.g. corrupt data)
)

// Result holds all information about a single file processing outcome.
type Result struct {
	Status    Status
	QVWPath   string // relative path shown to user
	QVSPath   string // relative output path (only for StatusOK)
	CharCount int    // character count of extracted script (only for StatusOK)
	Message   string // warning/error message (for StatusWarn and StatusErr)
}

var (
	okStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))  // green
	warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))  // yellow
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))  // red
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // dim
)

// Printer writes formatted output to w.
type Printer struct {
	w      io.Writer
	tty    bool
	dryRun bool

	okCount   int
	warnCount int
	errCount  int
}

// NewPrinter creates a Printer. tty enables color/spinner; dryRun appends [dry run].
func NewPrinter(w io.Writer, tty bool, dryRun bool) *Printer {
	return &Printer{w: w, tty: tty, dryRun: dryRun}
}

// FileResult prints a single per-file result line and updates internal counters.
func (p *Printer) FileResult(r Result) {
	switch r.Status {
	case StatusOK:
		p.okCount++
		p.printOK(r)
	case StatusWarn:
		p.warnCount++
		p.printWarn(r)
	case StatusErr:
		p.errCount++
		p.printErr(r)
	}
}

func (p *Printer) printOK(r Result) {
	sym := p.colorize("✓", okStyle)
	count := formatCount(r.CharCount)
	line := fmt.Sprintf("  %s  %s → %s  (%s chars)", sym, r.QVWPath, r.QVSPath, count)
	if p.dryRun {
		line += "  " + p.colorize("[dry run]", dimStyle)
	}
	fmt.Fprintln(p.w, line)
}

func (p *Printer) printWarn(r Result) {
	sym := p.colorize("⚠", warnStyle)
	line := fmt.Sprintf("  %s  %s  %s", sym, r.QVWPath, r.Message)
	if p.dryRun {
		line += "  " + p.colorize("[dry run]", dimStyle)
	}
	fmt.Fprintln(p.w, line)
}

func (p *Printer) printErr(r Result) {
	sym := p.colorize("✗", errStyle)
	line := fmt.Sprintf("  %s  %s  %s", sym, r.QVWPath, r.Message)
	if p.dryRun {
		line += "  " + p.colorize("[dry run]", dimStyle)
	}
	fmt.Fprintln(p.w, line)
}

// Summary prints the final summary line.
func (p *Printer) Summary() {
	total := p.okCount + p.warnCount + p.errCount

	ok := p.colorize(fmt.Sprintf("✓ %d", p.okCount), okStyle)
	warn := p.colorize(fmt.Sprintf("⚠ %d", p.warnCount), warnStyle)
	er := p.colorize(fmt.Sprintf("✗ %d", p.errCount), errStyle)
	counts := fmt.Sprintf("%s  %s  %s", ok, warn, er)

	var line string
	if p.dryRun {
		// total = all files attempted (OK + warn + err); matches spec: "N files would be extracted"
		line = fmt.Sprintf("  Dry run — %d files would be extracted  %s", total, counts)
	} else {
		line = fmt.Sprintf("  Extracted %d scripts   %s", p.okCount, counts)
	}
	fmt.Fprintln(p.w, line)
}

// UpdateSpinner prints a spinner line (non-TTY: no-op; TTY: overwrite current line).
// current is the 1-based index of the file being processed; total is the full count.
func (p *Printer) UpdateSpinner(current, total int) {
	if !p.tty {
		return
	}
	fmt.Fprintf(p.w, "\r  Extracting... %d/%d", current, total)
}

// ClearSpinner clears the spinner line (TTY only).
func (p *Printer) ClearSpinner() {
	if !p.tty {
		return
	}
	fmt.Fprintf(p.w, "\r%s\r", strings.Repeat(" ", 40))
}

// colorize applies the style only in TTY mode (colors disabled otherwise).
func (p *Printer) colorize(s string, style lipgloss.Style) string {
	if !p.tty {
		return s
	}
	return style.Render(s)
}

// formatCount formats an integer with comma thousands separators.
func formatCount(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
```

- [ ] **Step 2: Run UI tests**

Run: `go test ./internal/ui/... -v`
Expected: All PASS

- [ ] **Step 3: Commit**

```bash
git add internal/ui/output.go internal/ui/output_test.go
git commit -m "feat: implement terminal UI printer with lipgloss styling"
```

---

## Chunk 2: TTY Detection

### Task 3: Add TTY detection helper test

**Files:**
- Create: `internal/ui/tty_test.go`

- [ ] **Step 1: Write TTY detection test**

```go
package ui_test

import (
	"os"
	"testing"

	"github.com/your-org/qlik-script-extractor/internal/ui"
)

func TestIsTTY_NonTTY(t *testing.T) {
	// os.Stdin is not a TTY in test environments
	// We create a regular file and check it's not a TTY
	f, err := os.CreateTemp("", "tty-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	if ui.IsTTY(f) {
		t.Error("expected regular file to not be a TTY")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ui/... -run TestIsTTY -v`
Expected: FAIL — `IsTTY` not defined.

---

### Task 4: Implement IsTTY

**Files:**
- Create: `internal/ui/tty.go`

- [ ] **Step 1: Write tty.go**

```go
package ui

import (
	"os"

	"golang.org/x/term"
)

// IsTTY reports whether f is a terminal (TTY).
func IsTTY(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
```

- [ ] **Step 2: Add golang.org/x/term dependency**

> **Note:** `golang.org/x/term` is not listed in the spec's dependency table but is the standard stdlib-adjacent package for TTY detection on all platforms. It is a safe addition.

```bash
go get golang.org/x/term
go mod tidy
```

- [ ] **Step 3: Run TTY test**

Run: `go test ./internal/ui/... -run TestIsTTY -v`
Expected: PASS

- [ ] **Step 4: Run all UI tests**

Run: `go test ./internal/ui/... -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/tty.go internal/ui/tty_test.go go.mod go.sum
git commit -m "feat: add TTY detection helper"
```

---

### Task 5: Validate UI coverage and lint

- [ ] **Step 1: Check coverage**

Run: `go test ./internal/ui/... -coverprofile=coverage.out && go tool cover -func=coverage.out`
Expected: >80% total coverage for `internal/ui/`.

- [ ] **Step 2: Run linter**

Run: `make lint`
Expected: No errors. Fix any issues before proceeding.

- [ ] **Step 3: Commit lint fixes if any**

```bash
git add -p
git commit -m "fix: resolve linter warnings in ui package"
```

(Only create this commit if there are lint fixes.)

- [ ] **Step 4: Run full test suite**

Run: `make test`
Expected: All tests pass.
