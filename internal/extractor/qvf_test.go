package extractor_test

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/mattiasthalen/qlik-parser/internal/extractor"
)

// buildQVFFixture builds a fake QVF file containing one zlib stream
// that decompresses to the given JSON bytes, preceded by arbitrary junk.
func buildQVFFixture(t *testing.T, jsonPayload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, _ = w.Write(jsonPayload)
	_ = w.Close()

	// Prepend 64 bytes of junk (0xFF) so the scanner has to find the stream.
	junk := make([]byte, 64)
	for i := range junk {
		junk[i] = 0xFF
	}
	return append(junk, buf.Bytes()...)
}

func TestExtractScriptFromQVF_ValidScript(t *testing.T) {
	script := "///$tab Main\r\nSET ThousandSep=',';"
	payload, _ := json.Marshal(map[string]string{"qScript": script})
	data := buildQVFFixture(t, payload)

	f, err := os.CreateTemp("", "valid_*.qvf")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	_, _ = f.Write(data)
	_ = f.Close()

	got, err := extractor.ExtractScriptFromQVF(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != script {
		t.Errorf("expected %q, got %q", script, got)
	}
}

func TestExtractScriptFromQVF_NoQScript(t *testing.T) {
	// A valid zlib stream with JSON that has no qScript field.
	payload, _ := json.Marshal(map[string]string{"other": "value"})
	data := buildQVFFixture(t, payload)

	f, err := os.CreateTemp("", "noscript_*.qvf")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	_, _ = f.Write(data)
	_ = f.Close()

	_, err = extractor.ExtractScriptFromQVF(f.Name())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var noScript *extractor.NoScriptError
	if !errors.As(err, &noScript) {
		t.Errorf("expected *NoScriptError, got %T: %v", err, err)
	}
}

func TestExtractScriptFromQVF_FileNotFound(t *testing.T) {
	_, err := extractor.ExtractScriptFromQVF("/nonexistent/path.qvf")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if errors.As(err, new(*extractor.NoScriptError)) {
		t.Errorf("expected os-level error, got NoScriptError")
	}
}
