package ignore_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"janitor/internal/core/ignore"
)

func TestNewIgnoreMatcher(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "janitor_ignore_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test case 1: Non-existent ignore file
	nonExistentPath := filepath.Join(tempDir, ".nonexistentignore")
	matcher, err := ignore.NewIgnoreMatcher(nonExistentPath)
	assert.NoError(t, err)
	assert.NotNil(t, matcher)
	assert.False(t, matcher.Matches("any/path"), "Should not match anything if file is non-existent")

	// Test case 2: Valid ignore file with comments, empty lines, and patterns
	ignoreContent := `
# This is a comment

*.log
/temp/
!important.log
`
	validIgnorePath := filepath.Join(tempDir, ".janitorignore")
	err = os.WriteFile(validIgnorePath, []byte(ignoreContent), 0644)
	assert.NoError(t, err)

	matcher, err = ignore.NewIgnoreMatcher(validIgnorePath)
	assert.NoError(t, err)
	assert.NotNil(t, matcher)

	assert.True(t, matcher.Matches("foo.log"), "Should ignore .log files")
	assert.True(t, matcher.Matches("sub/dir/bar.log"), "Should ignore .log files in subdirectories")
	assert.True(t, matcher.Matches("temp/file.txt"), "Should ignore /temp/ directory")
	assert.True(t, matcher.Matches("temp/sub/file.txt"), "Should ignore /temp/ subdirectories")
	assert.False(t, matcher.Matches("important.log"), "Should not ignore important.log due to negation")
	assert.False(t, matcher.Matches("other.txt"), "Should not ignore other files")

	// Test case 3: Empty ignore file
	emptyIgnorePath := filepath.Join(tempDir, ".emptyignore")
	err = os.WriteFile(emptyIgnorePath, []byte(""), 0644)
	assert.NoError(t, err)

	matcher, err = ignore.NewIgnoreMatcher(emptyIgnorePath)
	assert.NoError(t, err)
	assert.NotNil(t, matcher)
	assert.False(t, matcher.Matches("any/path"), "Should not match anything if file is empty")
}

func TestNewIgnoreMatcherFromPatterns(t *testing.T) {
	// Test case 1: Empty patterns slice
	matcher, err := ignore.NewIgnoreMatcherFromPatterns([]string{})
	assert.NoError(t, err)
	assert.NotNil(t, matcher)
	assert.False(t, matcher.Matches("any/path"), "Should not match anything if patterns are empty")

	// Test case 2: Valid patterns slice
	patterns := []string{
		"*.tmp",
		"build/",
		"!src/main.tmp",
	}
	matcher, err = ignore.NewIgnoreMatcherFromPatterns(patterns)
	assert.NoError(t, err)
	assert.NotNil(t, matcher)

	t.Logf("Patterns for TestNewIgnoreMatcherFromPatterns (case 2): %v", patterns)
	// Log compiled patterns for debugging
	// if matcher != nil {
	// 	for i, p := range matcher.patterns {
	// 		t.Logf("  Compiled Pattern %d: %s", i, p.String())
	// 	}
	// 	for i, p := range matcher.negates {
	// 		t.Logf("  Compiled Negate %d: %s", i, p.String())
	// 	}
	// }

	assert.True(t, matcher.Matches("foo.tmp"), "Should ignore .tmp files")
	t.Logf("Testing path: %s, Matches: %t", "sub/dir/bar.tmp", matcher.Matches("sub/dir/bar.tmp"))
	assert.True(t, matcher.Matches("sub/dir/bar.tmp"), "Should ignore .tmp files in subdirectories")
	t.Logf("Testing path: %s, Matches: %t", "build/output.exe", matcher.Matches("build/output.exe"))
	assert.True(t, matcher.Matches("build/output.exe"), "Should ignore build/ directory")
	t.Logf("Testing path: %s, Matches: %t", "build/sub/file.txt", matcher.Matches("build/sub/file.txt"))
	assert.True(t, matcher.Matches("build/sub/file.txt"), "Should ignore build/ subdirectories")
	assert.False(t, matcher.Matches("src/main.tmp"), "Should not ignore src/main.tmp due to negation")
	assert.False(t, matcher.Matches("other.txt"), "Should not ignore other files")

	// Test case 3: Simple directory name heuristic
	patterns = []string{
		"node_modules", // Should be treated as **/node_modules/**
	}
	matcher, err = ignore.NewIgnoreMatcherFromPatterns(patterns)
	assert.NoError(t, err)
	assert.NotNil(t, matcher)

	t.Logf("Patterns for TestNewIgnoreMatcherFromPatterns (case 3): %v", patterns)
	// Log compiled patterns for debugging
	// if matcher != nil {
	// 	for i, p := range matcher.patterns {
	// 		t.Logf("  Compiled Pattern %d: %s", i, p.String())
	// 	}
	// 	for i, p := range matcher.negates {
	// 		t.Logf("  Compiled Negate %d: %s", i, p.String())
	// 	}
	// }

	t.Logf("Testing path: %s, Matches: %t", "node_modules/package.json", matcher.Matches("node_modules/package.json"))
	assert.True(t, matcher.Matches("node_modules/package.json"), "Should ignore node_modules content")
	t.Logf("Testing path: %s, Matches: %t", "project/node_modules/some_file.js", matcher.Matches("project/node_modules/some_file.js"))
	assert.True(t, matcher.Matches("project/node_modules/some_file.js"), "Should ignore node_modules in subdirectories")
	assert.False(t, matcher.Matches("my_node_modules"), "Should not ignore partial matches")
}

func TestMatches(t *testing.T) {
	// Test with a matcher created from patterns
	patterns := []string{
		"*.bak",
		"node_modules/",
		"!node_modules/package.json",
	}
	matcher, err := ignore.NewIgnoreMatcherFromPatterns(patterns)
	assert.NoError(t, err)

	t.Logf("Patterns for TestMatches: %v", patterns)
	// Log compiled patterns for debugging
	// if matcher != nil {
	// 	for i, p := range matcher.patterns {
	// 		t.Logf("  Compiled Pattern %d: %s", i, p.String())
	// 	}
	// 	for i, p := range matcher.negates {
	// 		t.Logf("  Compiled Negate %d: %s", i, p.String())
	// 	}
	// }

	t.Logf("Testing path: %s, Matches: %t", "file.bak", matcher.Matches("file.bak"))
	assert.True(t, matcher.Matches("file.bak"), "Should match .bak file")
	t.Logf("Testing path: %s, Matches: %t", "dir/file.bak", matcher.Matches("dir/file.bak"))
	assert.True(t, matcher.Matches("dir/file.bak"), "Should match .bak file in subdir")
	t.Logf("Testing path: %s, Matches: %t", "node_modules/some_lib/index.js", matcher.Matches("node_modules/some_lib/index.js"))
	assert.True(t, matcher.Matches("node_modules/some_lib/index.js"), "Should match node_modules content")
	t.Logf("Testing path: %s, Matches: %t", "node_modules/package.json", matcher.Matches("node_modules/package.json"))
	assert.False(t, matcher.Matches("node_modules/package.json"), "Should not match negated file")
	assert.False(t, matcher.Matches("src/file.go"), "Should not match non-ignored file")

	// Test with nil matcher
	var nilMatcher *ignore.IgnoreMatcher
	assert.False(t, nilMatcher.Matches("any/path"), "Nil matcher should not match")

	// Test with empty matcher (isEmpty = true)
	emptyMatcher, _ := ignore.NewIgnoreMatcherFromPatterns([]string{})
	assert.False(t, emptyMatcher.Matches("any/path"), "Empty matcher should not match")
}
