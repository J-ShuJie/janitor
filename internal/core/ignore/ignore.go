package ignore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"janitor/internal/logger"

	"github.com/gobwas/glob"
)

// IgnoreMatcher holds compiled glob patterns for ignore rules.
type IgnoreMatcher struct {
	patterns []glob.Glob
	negates  []glob.Glob // Patterns starting with '!'
}

// NewIgnoreMatcherFromPatterns creates an IgnoreMatcher from a slice of glob patterns.
func NewIgnoreMatcherFromPatterns(patterns []string) (*IgnoreMatcher, error) {
	var compiledPatterns []glob.Glob
	var compiledNegates []glob.Glob

	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" || strings.HasPrefix(p, "#") { // Ignore empty lines and comments
			continue
		}

		isNegated := false
		if strings.HasPrefix(p, "!") {
			isNegated = true
			p = p[1:] // Remove '!'
		}

		originalPattern := p // Keep original for logging/debugging

		// .gitignore semantics:
		// - Trailing slash matches directories and their contents.
		// - Leading slash anchors the pattern to the root.
		// - No slashes matches files or directories with that name anywhere.
		// - Wildcards like *.log match anywhere

		var patternsToCompile []string

		if strings.HasPrefix(originalPattern, "/") {
			// Anchored to root (e.g., "/temp/")
			trimmedPattern := strings.TrimPrefix(originalPattern, "/")
			if strings.HasSuffix(trimmedPattern, "/") {
				// Directory pattern
				patternsToCompile = append(patternsToCompile, strings.TrimSuffix(trimmedPattern, "/"))
				patternsToCompile = append(patternsToCompile, strings.TrimSuffix(trimmedPattern, "/")+"/**")
			} else {
				patternsToCompile = append(patternsToCompile, trimmedPattern)
			}
		} else if strings.HasSuffix(originalPattern, "/") {
			// Directory pattern anywhere (e.g., "build/")
			dirName := strings.TrimSuffix(originalPattern, "/")
			patternsToCompile = append(patternsToCompile, dirName)
			patternsToCompile = append(patternsToCompile, "*/"+dirName)
			patternsToCompile = append(patternsToCompile, "**/"+dirName)
			patternsToCompile = append(patternsToCompile, dirName+"/**")
			patternsToCompile = append(patternsToCompile, "*/"+dirName+"/**")
			patternsToCompile = append(patternsToCompile, "**/"+dirName+"/**")
		} else if strings.Contains(originalPattern, "/") {
			// Path pattern (e.g., "foo/bar")
			patternsToCompile = append(patternsToCompile, originalPattern)
			patternsToCompile = append(patternsToCompile, "**/"+originalPattern)
		} else if strings.Contains(originalPattern, "*") || strings.Contains(originalPattern, "?") {
			// Wildcard pattern (e.g., "*.log", "test?.txt")
			patternsToCompile = append(patternsToCompile, originalPattern)
			patternsToCompile = append(patternsToCompile, "*/"+originalPattern)
			patternsToCompile = append(patternsToCompile, "**/"+originalPattern)
		} else {
			// Simple name (e.g., "node_modules")
			patternsToCompile = append(patternsToCompile, originalPattern)
			patternsToCompile = append(patternsToCompile, "*/"+originalPattern)
			patternsToCompile = append(patternsToCompile, "**/"+originalPattern)
			patternsToCompile = append(patternsToCompile, originalPattern+"/**")
			patternsToCompile = append(patternsToCompile, "*/"+originalPattern+"/**")
			patternsToCompile = append(patternsToCompile, "**/"+originalPattern+"/**")
		}

		for _, pToCompile := range patternsToCompile {
			g, err := glob.Compile(pToCompile, '/') // Use '/' as separator for consistency
			if err != nil {
				logger.Log.Warn("Failed to compile ignore pattern", "pattern", originalPattern, "compiled_pattern", pToCompile, "error", err)
				continue
			}
			if isNegated {
				compiledNegates = append(compiledNegates, g)
			} else {
				compiledPatterns = append(compiledPatterns, g)
			}
		}
	}

	return &IgnoreMatcher{
		patterns: compiledPatterns,
		negates:  compiledNegates,
	}, nil
}
// NewIgnoreMatcher creates an IgnoreMatcher by reading patterns from a .janitorignore file.
func NewIgnoreMatcher(filePath string) (*IgnoreMatcher, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &IgnoreMatcher{}, nil // No file, no patterns, not an error
		}
		return nil, err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		patterns = append(patterns, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return NewIgnoreMatcherFromPatterns(patterns)
}

// Matches checks if the given path matches any of the ignore patterns.
// Negated patterns take precedence.
func (m *IgnoreMatcher) Matches(path string) bool {
	if m == nil {
		return false
	}
	// Normalize path for matching
	path = filepath.ToSlash(filepath.Clean(path))

	// Check negated patterns first
	for _, negate := range m.negates {
		if negate.Match(path) {
			return false // Explicitly not ignored
		}
	}

	// Check regular patterns
	for _, pattern := range m.patterns {
		if pattern.Match(path) {
			return true // Ignored
		}
	}

	return false
}
