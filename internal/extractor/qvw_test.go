package extractor_test

import (
	"bytes"
	"compress/zlib"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/mattiasthalen/qlik-parser/internal/extractor"
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
	if strings.Contains(err.Error(), "zlib: zlib:") {
		t.Errorf("error message has double zlib: prefix: %v", err)
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

func TestExtractScript_LongScriptNotTruncated(t *testing.T) {
	const scriptLen = 200_000
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	payload := make([]byte, scriptLen+10) // +10 for end marker
	copy(payload, []byte("///"))
	for i := 3; i < scriptLen; i++ {
		payload[i] = 'X'
	}
	// end marker: \n followed by two \x00 bytes
	copy(payload[scriptLen:], []byte{'\n', 0x00, 0x00})
	_, _ = w.Write(payload)
	_ = w.Close()
	header := make([]byte, 23)
	data := append(header, buf.Bytes()...)

	f, err := os.CreateTemp("", "long_script_test_*.qvw")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	_, _ = f.Write(data)
	_ = f.Close()

	script, err := extractor.ExtractScript(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(script) < scriptLen-1 {
		t.Errorf("expected full script (~%d chars), got %d — possible truncation", scriptLen, len(script))
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
