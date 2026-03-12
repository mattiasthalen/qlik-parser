package extractor

import (
	"os"
	"path/filepath"
)

// Artifact is a named file payload to be written into an output directory.
type Artifact struct {
	Name    string // filename, e.g. "script.qvs"
	Content []byte
}

// ResolveOutputDir computes the output folder path for a source file.
//
//   - inputPath: absolute path to the source file (.qvw or .qvf)
//   - sourceDir: the --source directory
//   - outDir:    the --out directory; empty string or equal to sourceDir → alongside mode
//
// The folder name is the full source filename including extension (e.g. "sales.qvw").
// No trailing slash is added (standard Go path convention).
func ResolveOutputDir(inputPath, sourceDir, outDir string) string {
	base := filepath.Base(inputPath) // e.g. "sales.qvw"

	if outDir == "" || outDir == sourceDir {
		return filepath.Join(filepath.Dir(inputPath), base)
	}

	rel, err := filepath.Rel(sourceDir, filepath.Dir(inputPath))
	if err != nil {
		return filepath.Join(outDir, base)
	}
	return filepath.Join(outDir, rel, base)
}

// WriteArtifacts writes each artifact into outDir. In dry-run mode it is a no-op.
// Intermediate directories are created automatically.
// Fail-fast: returns the first error encountered without attempting remaining writes.
func WriteArtifacts(outDir string, artifacts []Artifact, dryRun bool) error {
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	for _, a := range artifacts {
		if err := os.WriteFile(filepath.Join(outDir, a.Name), a.Content, 0644); err != nil {
			return err
		}
	}
	return nil
}
