//go:build ignore

package main

// Run: go run internal/extractor/testdata/gen/main.go
// Generates binary .qvw fixture files for tests.

import (
	"bytes"
	"compress/zlib"
	"os"
)

func makeQVW(payload []byte) []byte {
	header := make([]byte, 23) // arbitrary placeholder header
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(payload)
	w.Close()
	return append(header, buf.Bytes()...)
}

func write(path string, data []byte) {
	if err := os.WriteFile(path, data, 0644); err != nil {
		panic(err)
	}
}

func main() {
	dir := "internal/extractor/testdata/fixtures"

	// valid.qvw: has a script between /// and end marker
	script := []byte("///\nLOAD * FROM table.csv;\n")
	payload := append(script, []byte("\r\n\x00\x00\x00")...)
	write(dir+"/valid.qvw", makeQVW(payload))

	// no_script.qvw: no /// marker
	write(dir+"/no_script.qvw", makeQVW([]byte("some binary data without triple slash")))

	// no_end_marker.qvw: /// found but no end marker — script is full 100k region
	longScript := make([]byte, 0, 200)
	longScript = append(longScript, []byte("///\nLOAD * FROM big_table;")...)
	write(dir+"/no_end_marker.qvw", makeQVW(longScript))

	// invalid_zlib.qvw: valid header size but garbage compressed data
	garbage := make([]byte, 50)
	for i := range garbage {
		garbage[i] = 0xFF
	}
	header := make([]byte, 23)
	write(dir+"/invalid_zlib.qvw", append(header, garbage...))

	// too_short.qvw: fewer than 23 bytes
	write(dir+"/too_short.qvw", []byte("short"))

	// invalid_utf8.qvw: script contains bytes that are invalid UTF-8
	utf8Payload := []byte("///\nLOAD \xFF\xFE * FROM table;\n\r\n\x00\x00")
	write(dir+"/invalid_utf8.qvw", makeQVW(utf8Payload))
}
