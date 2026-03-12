package ui_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattiasthalen/qlik-script-extractor/internal/ui"
)

func newTestPrinter(buf *bytes.Buffer) *ui.Printer {
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
	p := ui.NewPrinter(buf, false, true)
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

func TestClearSpinner_PaddingMatchesSpinnerTextWidth(t *testing.T) {
	buf := &bytes.Buffer{}
	p := ui.NewPrinter(buf, true, false) // TTY=true to enable spinner

	// Write a spinner line and measure its visible length (everything after leading \r)
	p.UpdateSpinner(1, 1)
	spinnerOutput := buf.String()
	buf.Reset()
	visibleLen := len(strings.TrimPrefix(spinnerOutput, "\r"))

	// ClearSpinner must pad with exactly as many spaces as the visible spinner text
	p.ClearSpinner()
	clearOut := buf.String()
	spaces := strings.Count(clearOut, " ")
	if spaces != visibleLen {
		t.Errorf("ClearSpinner padding = %d spaces, want %d (spinner text width)", spaces, visibleLen)
	}
}
