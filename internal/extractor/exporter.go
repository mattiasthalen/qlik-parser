package extractor

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolveOutputPath computes the destination path for the extracted script.
//
//   - inputPath: absolute path to the source file (.qvw or .qvf)
//   - sourceDir: the --source directory (used to compute relative path)
//   - outDir:    the --out directory; empty string or equal to sourceDir → alongside mode
//
// Output extensions:
//   - .qvw → .qvs
//   - .qvf → .qvf.qvs  (double-extension avoids collision with same-named .qvw output)
func ResolveOutputPath(inputPath, sourceDir, outDir string) string {
	ext := filepath.Ext(inputPath)
	var outExt string
	switch ext {
	case ".qvf":
		outExt = ".qvf.qvs"
	default:
		outExt = ".qvs"
	}
	base := strings.TrimSuffix(filepath.Base(inputPath), ext) + outExt

	if outDir == "" || outDir == sourceDir {
		return filepath.Join(filepath.Dir(inputPath), base)
	}

	rel, err := filepath.Rel(sourceDir, filepath.Dir(inputPath))
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
