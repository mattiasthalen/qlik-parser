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

// makeQVFFile marshals payload as JSON, builds a fake QVF binary via buildQVFFixture,
// and writes it to a temp file. Returns the file path.
func makeQVFFile(t *testing.T, payload map[string]any) string {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	data := buildQVFFixture(t, b)
	f, err := os.CreateTemp(t.TempDir(), "*.qvf")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	_, _ = f.Write(data)
	_ = f.Close()
	return f.Name()
}

func TestParseQVF_Script(t *testing.T) {
	path := makeQVFFile(t, map[string]any{
		"qScript": "LOAD * FROM t;",
	})
	got, err := extractor.ParseQVF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Script != "LOAD * FROM t;" {
		t.Errorf("Script = %q, want %q", got.Script, "LOAD * FROM t;")
	}
}

func TestParseQVF_Measures(t *testing.T) {
	path := makeQVFFile(t, map[string]any{
		"qInfo":    map[string]any{"qId": "abc", "qType": "measure"},
		"qMeasure": map[string]any{"qLabel": "Sales", "qDef": "Sum(Amount)", "qTags": []string{"tag1"}},
		"qMetaDef": map[string]any{"description": "Total sales"},
	})
	got, err := extractor.ParseQVF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Measures) != 1 {
		t.Fatalf("expected 1 measure, got %d", len(got.Measures))
	}
	m := got.Measures[0]
	if m.ID != "abc" || m.Label != "Sales" || m.Def != "Sum(Amount)" {
		t.Errorf("unexpected measure: %+v", m)
	}
	if len(m.Tags) != 1 || m.Tags[0] != "tag1" {
		t.Errorf("unexpected tags: %v", m.Tags)
	}
	if m.Description != "Total sales" {
		t.Errorf("unexpected description: %q", m.Description)
	}
}

func TestParseQVF_Dimensions(t *testing.T) {
	path := makeQVFFile(t, map[string]any{
		"qInfo":    map[string]any{"qId": "dim1", "qType": "dimension"},
		"qDim":     map[string]any{"qFieldDefs": []string{"Country"}, "qFieldLabels": []string{""}},
		"qMetaDef": map[string]any{"title": "Country", "description": "", "tags": []string{}},
	})
	got, err := extractor.ParseQVF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Dimensions) != 1 {
		t.Fatalf("expected 1 dimension, got %d", len(got.Dimensions))
	}
	d := got.Dimensions[0]
	if d.ID != "dim1" || d.Label != "Country" {
		t.Errorf("unexpected dimension: %+v", d)
	}
	if len(d.Fields) != 1 || d.Fields[0] != "Country" {
		t.Errorf("unexpected fields: %v", d.Fields)
	}
}

func TestParseQVF_Variables(t *testing.T) {
	path := makeQVFFile(t, map[string]any{
		"qId": "user_variablelist",
		"qEntryList": []map[string]any{
			{
				"qInfo": map[string]any{"qId": "uuid-1", "qType": "variable"},
				"qData": map[string]any{"qName": "vDaysBack", "qComment": "lookback", "qValue": 30},
			},
		},
	})
	got, err := extractor.ParseQVF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Variables) != 1 {
		t.Fatalf("expected 1 variable, got %d", len(got.Variables))
	}
	v := got.Variables[0]
	if v.ID != "uuid-1" || v.Name != "vDaysBack" || v.Comment != "lookback" {
		t.Errorf("unexpected variable: %+v", v)
	}
	// Value must be raw JSON — not re-encoded (number stays number)
	if string(v.Value) != "30" {
		t.Errorf("Value = %s, want 30", string(v.Value))
	}
}

func TestParseQVF_EmptyArtifactLists(t *testing.T) {
	// A file with a script block only — Measures/Dimensions/Variables should be non-nil empty slices
	path := makeQVFFile(t, map[string]any{"qScript": "LOAD 1;"})
	got, err := extractor.ParseQVF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Measures == nil || got.Dimensions == nil || got.Variables == nil {
		t.Error("expected non-nil empty slices for missing artifact types")
	}
}

func TestParseQVF_EmptyScript(t *testing.T) {
	// ParseQVF itself never returns NoScriptError — Script field is just empty.
	path := makeQVFFile(t, map[string]any{"other": "data"})
	got, err := extractor.ParseQVF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Script != "" {
		t.Errorf("expected empty Script, got %q", got.Script)
	}
}
