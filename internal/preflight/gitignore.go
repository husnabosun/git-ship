package preflight

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
)

//go:embed templates/*.gitignore
var templateFS embed.FS

// extensionToTemplate maps a file extension to the template file that best
// fits it. Extend this map as new ecosystems are supported.
var extensionToTemplate = map[string]string{
	".go":  "Go",
	".py":  "Python",
	".js":  "Node",
	".ts":  "Node",
	".jsx": "Node",
	".tsx": "Node",
}

// skipDirs are never walked when detecting the ecosystem or scanning files —
// they are either huge, irrelevant, or already-ignored build artifacts.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	".venv":        true,
	"venv":         true,
	"__pycache__":  true,
}

// DetectEcosystem walks dir (one level deep is usually enough for a fresh
// project) and returns the template name whose extension appears most often.
// Returns "Generic" if nothing recognizable is found.
func DetectEcosystem(dir string) string {
	counts := map[string]int{}

	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // best-effort scan, ignore unreadable entries
		}
		if d.IsDir() {
			if path != dir && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(d.Name())
		if tmpl, ok := extensionToTemplate[ext]; ok {
			counts[tmpl]++
		}
		return nil
	})

	best, bestCount := "Generic", 0
	for tmpl, c := range counts {
		if c > bestCount {
			best, bestCount = tmpl, c
		}
	}
	return best
}

// LoadTemplate returns the contents of an embedded .gitignore template by name.
func LoadTemplate(name string) (string, error) {
	data, err := templateFS.ReadFile("templates/" + name + ".gitignore")
	if err != nil {
		// Fall back to Generic if an unexpected name is requested.
		fallback, fallbackErr := templateFS.ReadFile("templates/Generic.gitignore")
		return string(fallback), fallbackErr
	}
	return string(data), nil
}

// sensitivePatterns are the entries we consider "important enough to flag"
// if they're missing from an existing .gitignore.
var sensitivePatterns = []string{
	".env",
	"node_modules/",
	"__pycache__/",
	"*.pem",
	"*.key",
	".DS_Store",
}

// MissingPatterns compares an existing .gitignore's content against
// sensitivePatterns and returns the ones that appear to be absent.
func MissingPatterns(existingContent string) []string {
	var missing []string
	for _, pattern := range sensitivePatterns {
		if !strings.Contains(existingContent, pattern) {
			missing = append(missing, pattern)
		}
	}
	return missing
}
