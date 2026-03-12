package extractor

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// qvfPayload is used to unmarshal only the qScript field from a QVF JSON block.
type qvfPayload struct {
	QScript string `json:"qScript"`
}

// ExtractScriptFromQVF reads a .qvf file and returns the embedded load script.
//
// A .qvf file is a proprietary binary container holding multiple zlib-compressed
// blocks. This function scans the file for zlib stream candidates (CMF byte 0x78
// followed by a valid FLG byte), decompresses each, and JSON-unmarshals to find
// the block containing a non-empty "qScript" field.
//
// Errors:
//   - os read error if the file cannot be read
//   - *NoScriptError if no block with a qScript field is found
func ExtractScriptFromQVF(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}

	// Valid zlib FLG bytes for CMF=0x78 (deflate, window size 32KB):
	// The pair (CMF*256+FLG) must be divisible by 31.
	validFLG := map[byte]bool{0x01: true, 0x5E: true, 0x9C: true, 0xDA: true}

	for i := 0; i < len(data)-1; i++ {
		if data[i] != 0x78 || !validFLG[data[i+1]] {
			continue
		}
		r, err := zlib.NewReader(bytes.NewReader(data[i:]))
		if err != nil {
			continue
		}
		decompressed, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			continue
		}
		// Some QVF blocks have a trailing null byte; strip it before JSON parsing.
		trimmed := bytes.TrimRight(decompressed, "\x00")
		var payload qvfPayload
		if err := json.Unmarshal(trimmed, &payload); err != nil {
			continue
		}
		if payload.QScript != "" {
			return payload.QScript, nil
		}
	}

	return "", &NoScriptError{Path: path}
}
