package extractor

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const headerSize = 23

// NoScriptError is returned when no /// marker is found in the decompressed data.
type NoScriptError struct {
	Path string
}

func (e *NoScriptError) Error() string {
	return fmt.Sprintf("%s: no script found", e.Path)
}

// IsNoScript reports whether err is a *NoScriptError and sets target if so.
func IsNoScript(err error, target **NoScriptError) bool {
	return errors.As(err, target)
}

// ExtractScript reads a .qvw file, decompresses its body, and returns the
// embedded load script as a UTF-8 string.
//
// Errors:
//   - "file too short" if file is < 23 bytes
//   - zlib error on decompression failure
//   - *NoScriptError if no /// marker is found
func ExtractScript(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	if len(data) < headerSize {
		return "", fmt.Errorf("%s: file too short", path)
	}

	compressed := data[headerSize:]
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	defer func() { _ = r.Close() }()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}

	return extractFromBytes(path, decompressed)
}

// extractFromBytes extracts the script from raw decompressed bytes.
func extractFromBytes(path string, data []byte) (string, error) {
	marker := []byte("///")
	scriptStart := bytes.Index(data, marker)
	if scriptStart < 0 {
		return "", &NoScriptError{Path: path}
	}

	region := data[scriptStart:]

	scriptBytes := trimAtEndMarker(region)
	return strings.ToValidUTF8(string(scriptBytes), "\uFFFD"), nil
}

// trimAtEndMarker finds the end of the script region:
// a newline (\r\n or \n) followed by two or more \x00 bytes.
// Returns region up to (not including) the trailing newline.
func trimAtEndMarker(region []byte) []byte {
	for i := 0; i < len(region)-2; i++ {
		if region[i] == '\n' {
			nlStart := i
			if i > 0 && region[i-1] == '\r' {
				nlStart = i - 1
			}
			j := i + 1
			for j < len(region) && region[j] == 0x00 {
				j++
			}
			if j-i-1 >= 2 {
				return region[:nlStart]
			}
		}
	}
	return region
}
