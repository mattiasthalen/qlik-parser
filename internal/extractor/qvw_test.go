package extractor_test

import (
	"bytes"
	"compress/zlib"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/mattiasthalen/qlik-script-extractor/internal/extractor"
)

const fixturesDir = "testdata/fixtures"

func TestExtractScript_ValidFile(t *testing.T) {
	script, err := extractor.ExtractScript(fixturesDir + "/valid.qvw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(script, "///") {
		t.Errorf("expected script to start with ///, got: %q", script[:min(20, len(script))])
	}
	if !strings.Contains(script, "LOAD * FROM table.csv") {
		t.Errorf("expected script content, got: %q", script)
	}
}

func TestExtractScript_TooShort(t *testing.T) {
	_, err := extractor.ExtractScript(fixturesDir + "/too_short.qvw")
	if err == nil {
		t.Fatal("expected error for too-short file, got nil")
	}
	if !strings.Contains(err.Error(), "file too short") {
		t.Errorf("expected 'file too short' error, got: %v", err)
	}
}

func TestExtractScript_InvalidZlib(t *testing.T) {
	_, err := extractor.ExtractScript(fixturesDir + "/invalid_zlib.qvw")
	if err == nil {
		t.Fatal("expected error for invalid zlib, got nil")
	}
}

func TestExtractScript_NoScriptMarker(t *testing.T) {
	_, err := extractor.ExtractScript(fixturesDir + "/no_script.qvw")
	if err == nil {
		t.Fatal("expected error for missing /// marker, got nil")
	}
	var noScript *extractor.NoScriptError
	if !errors.As(err, &noScript) {
		t.Errorf("expected NoScriptError, got: %T %v", err, err)
	}
}

func TestExtractScript_NoEndMarker(t *testing.T) {
	script, err := extractor.ExtractScript(fixturesDir + "/no_end_marker.qvw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(script, "///") {
		t.Errorf("expected script to start with ///, got: %q", script)
	}
}

func TestExtractScript_TruncatesAt100k(t *testing.T) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	payload := make([]byte, 200_100)
	copy(payload, []byte("///"))
	for i := 3; i < len(payload); i++ {
		payload[i] = 'X'
	}
	w.Write(payload)
	w.Close()
	header := make([]byte, 23)
	data := append(header, buf.Bytes()...)

	f, err := os.CreateTemp("", "truncate_test_*.qvw")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Write(data)
	f.Close()

	script, err := extractor.ExtractScript(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(script) > 100_000 {
		t.Errorf("expected script truncated to 100,000 bytes, got %d", len(script))
	}
}

func TestExtractScript_InvalidUTF8(t *testing.T) {
	script, err := extractor.ExtractScript(fixturesDir + "/invalid_utf8.qvw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(script, "\uFFFD") {
		t.Errorf("expected replacement character in output for invalid UTF-8, got: %q", script)
	}
}

func TestIsNoScript(t *testing.T) {
	err := &extractor.NoScriptError{Path: "foo.qvw"}
	var target *extractor.NoScriptError
	if !extractor.IsNoScript(err, &target) {
		t.Error("expected IsNoScript to return true")
	}
	if target.Path != "foo.qvw" {
		t.Errorf("expected Path=foo.qvw, got %s", target.Path)
	}
}
