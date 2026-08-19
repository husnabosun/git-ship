package preflight

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// suspiciousNames are filename patterns that commonly hold secrets.
// This is intentionally a simple name-based heuristic, not a content scanner —
// a full secret-scanning engine (entropy analysis, regex rules, etc.) is out
// of scope for a pre-flight check and belongs in a dedicated tool.
var suspiciousNames = []string{
	".env",
	".env.local",
	"id_rsa",
	"id_ed25519",
}

var suspiciousSuffixes = []string{
	".pem",
	".key",
}

// const to flag unusually large files that may have been staged by mistake.
const largeFileThresholdBytes = 5 * 1024 * 1024 // 5MB

// ScanForSensitiveFiles walks dir and returns a list of relative paths that
// look like they might contain secrets or shouldn't be committed.
func ScanForSensitiveFiles(dir string) []string {
	var found []string

	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != dir && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		name := d.Name()
		rel, _ := filepath.Rel(dir, path)

		for _, n := range suspiciousNames {
			if name == n {
				found = append(found, rel)
				return nil
			}
		}
		for _, suf := range suspiciousSuffixes {
			if strings.HasSuffix(name, suf) {
				found = append(found, rel)
				return nil
			}
		}

		if info, err := d.Info(); err == nil && info.Size() > largeFileThresholdBytes {
			found = append(found, rel+" (large file, "+formatSize(info.Size())+")")
		}

		return nil
	})

	return found
}

func formatSize(bytes int64) string {
	mb := float64(bytes) / (1024 * 1024)
	return fmt.Sprintf("%.1fMB", mb)
}
