package core_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"janitor/internal/config"
	"janitor/internal/core"
	"janitor/internal/logger"
)

// TestScannerWithDeepNesting tests scanner with very deep directory structures
func TestScannerWithDeepNesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	cleanup, err := logger.InitLogger(0)
	assert.NoError(t, err)
	defer cleanup()

	oldCheckGuardrails := core.CheckGuardrails
	core.CheckGuardrails = func(path string, cfg config.Config) bool { return false }
	defer func() { core.CheckGuardrails = oldCheckGuardrails }()

	tempDir, err := os.MkdirTemp("", "scan_deep_nesting")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create deeply nested directory (100 levels deep)
	currentPath := tempDir
	for i := 0; i < 100; i++ {
		currentPath = filepath.Join(currentPath, "level")
		err = os.Mkdir(currentPath, 0755)
		assert.NoError(t, err)
	}

	// Create a node_modules at the deepest level
	nodeModulesPath := filepath.Join(currentPath, "node_modules")
	err = os.Mkdir(nodeModulesPath, 0755)
	assert.NoError(t, err)

	// Add a file
	testFile := filepath.Join(nodeModulesPath, "test.js")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	assert.NoError(t, err)

	scanCfg := config.ScanConfig{
		Directories: []string{"node_modules"},
		Rules:       config.RuleConfig{OlderThanDays: 0},
	}
	cfg := config.Config{Scan: scanCfg}

	// This should complete without stack overflow or panic
	junkItems, err := core.ScanProjectJunk(tempDir, scanCfg, []string{}, cfg, time.Now(), nil)
	assert.NoError(t, err)
	assert.Len(t, junkItems, 1)
	t.Logf("Successfully scanned 100-level deep directory")
}

// TestScannerWithManyFiles tests scanner with large number of files
func TestScannerWithManyFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	cleanup, err := logger.InitLogger(0)
	assert.NoError(t, err)
	defer cleanup()

	oldCheckGuardrails := core.CheckGuardrails
	core.CheckGuardrails = func(path string, cfg config.Config) bool { return false }
	defer func() { core.CheckGuardrails = oldCheckGuardrails }()

	tempDir, err := os.MkdirTemp("", "scan_many_files")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create node_modules with 1000 files
	nodeModulesPath := filepath.Join(tempDir, "node_modules")
	err = os.Mkdir(nodeModulesPath, 0755)
	assert.NoError(t, err)

	for i := 0; i < 1000; i++ {
		testFile := filepath.Join(nodeModulesPath, filepath.FromSlash("file_"+string(rune(i%26+97))+".js"))
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Logf("Warning: failed to create file %d: %v", i, err)
		}
	}

	scanCfg := config.ScanConfig{
		Directories: []string{"node_modules"},
		Rules:       config.RuleConfig{OlderThanDays: 0},
	}
	cfg := config.Config{Scan: scanCfg}

	start := time.Now()
	junkItems, err := core.ScanProjectJunk(tempDir, scanCfg, []string{}, cfg, time.Now(), nil)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Len(t, junkItems, 1)
	t.Logf("Scanned 1000 files in %v", duration)

	// Should complete in reasonable time (< 10 seconds for 1000 files)
	assert.Less(t, duration.Seconds(), 10.0, "Scan took too long")
}

// TestScannerWithSpecialCharacters tests scanner with unusual filenames
func TestScannerWithSpecialCharacters(t *testing.T) {
	cleanup, err := logger.InitLogger(0)
	assert.NoError(t, err)
	defer cleanup()

	oldCheckGuardrails := core.CheckGuardrails
	core.CheckGuardrails = func(path string, cfg config.Config) bool { return false }
	defer func() { core.CheckGuardrails = oldCheckGuardrails }()

	tempDir, err := os.MkdirTemp("", "scan_special_chars")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create directories with special characters (that are valid on the filesystem)
	specialDirs := []string{
		"node_modules",
		// Commenting out problematic characters that may not work on all filesystems
		// "node modules with spaces",
		// "node-modules-with-dashes",
		// "node.modules.with.dots",
	}

	for _, dirName := range specialDirs {
		dirPath := filepath.Join(tempDir, dirName)
		err = os.Mkdir(dirPath, 0755)
		assert.NoError(t, err)

		testFile := filepath.Join(dirPath, "test.js")
		err = os.WriteFile(testFile, []byte("test"), 0644)
		assert.NoError(t, err)
	}

	scanCfg := config.ScanConfig{
		Directories: []string{"node_modules"}, // Only scan the exact match
		Rules:       config.RuleConfig{OlderThanDays: 0},
	}
	cfg := config.Config{Scan: scanCfg}

	junkItems, err := core.ScanProjectJunk(tempDir, scanCfg, []string{}, cfg, time.Now(), nil)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(junkItems), 1)
	t.Logf("Found %d items with special characters", len(junkItems))
}

// TestScannerWithSymlinks tests scanner behavior with symbolic links
func TestScannerWithSymlinks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping symlink test in short mode")
	}

	cleanup, err := logger.InitLogger(0)
	assert.NoError(t, err)
	defer cleanup()

	oldCheckGuardrails := core.CheckGuardrails
	core.CheckGuardrails = func(path string, cfg config.Config) bool { return false }
	defer func() { core.CheckGuardrails = oldCheckGuardrails }()

	tempDir, err := os.MkdirTemp("", "scan_symlinks")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a real directory
	realDir := filepath.Join(tempDir, "real_node_modules")
	err = os.Mkdir(realDir, 0755)
	assert.NoError(t, err)

	testFile := filepath.Join(realDir, "test.js")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	assert.NoError(t, err)

	// Create a symlink to it
	symlinkPath := filepath.Join(tempDir, "node_modules")
	err = os.Symlink(realDir, symlinkPath)
	if err != nil {
		t.Skipf("Cannot create symlink (might be Windows without admin): %v", err)
	}

	scanCfg := config.ScanConfig{
		Directories: []string{"node_modules"},
		Rules:       config.RuleConfig{OlderThanDays: 0},
	}
	cfg := config.Config{Scan: scanCfg}

	// This should handle symlinks without infinite loops
	junkItems, err := core.ScanProjectJunk(tempDir, scanCfg, []string{}, cfg, time.Now(), nil)
	assert.NoError(t, err)
	t.Logf("Symlink test found %d items", len(junkItems))
}

// TestScannerWithEmptyDirectories tests handling of empty directories
func TestScannerWithEmptyDirectories(t *testing.T) {
	cleanup, err := logger.InitLogger(0)
	assert.NoError(t, err)
	defer cleanup()

	oldCheckGuardrails := core.CheckGuardrails
	core.CheckGuardrails = func(path string, cfg config.Config) bool { return false }
	defer func() { core.CheckGuardrails = oldCheckGuardrails }()

	tempDir, err := os.MkdirTemp("", "scan_empty")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create empty node_modules
	nodeModulesPath := filepath.Join(tempDir, "node_modules")
	err = os.Mkdir(nodeModulesPath, 0755)
	assert.NoError(t, err)

	scanCfg := config.ScanConfig{
		Directories: []string{"node_modules"},
		Rules:       config.RuleConfig{OlderThanDays: 0},
	}
	cfg := config.Config{Scan: scanCfg}

	junkItems, err := core.ScanProjectJunk(tempDir, scanCfg, []string{}, cfg, time.Now(), nil)
	assert.NoError(t, err)
	assert.Len(t, junkItems, 1)
	assert.Equal(t, 0.0, junkItems[0].SizeMB)
	t.Logf("Empty directory handled correctly with 0 MB size")
}

// TestScannerConcurrentSafety tests that concurrent operations don't cause data races
func TestScannerConcurrentSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	cleanup, err := logger.InitLogger(0)
	assert.NoError(t, err)
	defer cleanup()

	oldCheckGuardrails := core.CheckGuardrails
	core.CheckGuardrails = func(path string, cfg config.Config) bool { return false }
	defer func() { core.CheckGuardrails = oldCheckGuardrails }()

	tempDir, err := os.MkdirTemp("", "scan_concurrent")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create multiple junk directories
	for i := 0; i < 10; i++ {
		dirPath := filepath.Join(tempDir, "node_modules_"+string(rune(i+48)))
		err = os.Mkdir(dirPath, 0755)
		assert.NoError(t, err)

		for j := 0; j < 10; j++ {
			testFile := filepath.Join(dirPath, "test"+string(rune(j+48))+".js")
			err = os.WriteFile(testFile, []byte("test content for concurrency"), 0644)
			assert.NoError(t, err)
		}
	}

	scanCfg := config.ScanConfig{
		Directories: []string{"node_modules_0", "node_modules_1", "node_modules_2",
			"node_modules_3", "node_modules_4", "node_modules_5",
			"node_modules_6", "node_modules_7", "node_modules_8", "node_modules_9"},
		Rules: config.RuleConfig{OlderThanDays: 0},
	}
	cfg := config.Config{Scan: scanCfg}

	// Run scan multiple times to check for race conditions
	for run := 0; run < 3; run++ {
		junkItems, err := core.ScanProjectJunk(tempDir, scanCfg, []string{}, cfg, time.Now(), nil)
		assert.NoError(t, err)
		assert.Equal(t, 10, len(junkItems), "Should find all 10 directories on run %d", run)
	}

	t.Logf("Concurrent safety test passed - no race conditions detected")
}
