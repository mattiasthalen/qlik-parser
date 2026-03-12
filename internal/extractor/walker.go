package extractor

import (
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/rs/zerolog/log"
)

// Walk recursively collects all *.qvw and *.qvf file paths under root.
// Returns sorted paths and a slice of warning messages for unreadable directories.
// Symlinks are not followed.
func Walk(root string) (paths []string, warns []string) {
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			warns = append(warns, path+": "+err.Error())
			log.Warn().Str("path", path).Err(err).Msg("skipping unreadable path")
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip symlinks to directories (WalkDir does not follow them by default,
		// but symlinks to files would still be walked — skip those too).
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		ext := filepath.Ext(path)
		if !d.IsDir() && (ext == ".qvw" || ext == ".qvf") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		warns = append(warns, root+": "+err.Error())
	}
	sort.Strings(paths)
	return paths, warns
}
