package extractor

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolveOutputPath computes the destination .qvs path for a given .qvw.
//
//   - qvwPath:   absolute path to the source .qvw file
//   - sourceDir: the --source directory (used to compute relative path)
//   - outDir:    the --out directory; empty string or equal to sourceDir → alongside mode
func ResolveOutputPath(qvwPath, sourceDir, outDir string) string {
	base := strings.TrimSuffix(filepath.Base(qvwPath), ".qvw") + ".qvs"

	if outDir == "" || outDir == sourceDir {
		return filepath.Join(filepath.Dir(qvwPath), base)
	}

	rel, err := filepath.Rel(sourceDir, filepath.Dir(qvwPath))
	if err != nil {
		return filepath.Join(outDir, base)
	}
	return filepath.Join(outDir, rel, base)
}

// WriteScript writes script to outPath. In dry-run mode it is a no-op.
// Intermediate directories are created automatically.
func WriteScript(outPath, script string, dryRun bool) error {
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(outPath, []byte(script), 0644)
}
