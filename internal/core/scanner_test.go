package core_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"janitor/internal/config"
	"janitor/internal/core"
	"janitor/internal/logger"
)

func createTempFile(t *testing.T, dir, name string, size int64, modTime time.Time) string {
	filePath := filepath.Join(dir, name)
	content := make([]byte, size)
	err := os.WriteFile(filePath, content, 0644)
	assert.NoError(t, err)

	// Attempt to set modification time
	err = os.Chtimes(filePath, modTime, modTime)
	assert.NoError(t, err)

	// Verify the modification time after setting, and log it for debugging
	fileInfo, err := os.Stat(filePath)
	assert.NoError(t, err)
	// Normalize the read time to UTC and truncate to day for consistent comparison
	readModTimeNormalized := fileInfo.ModTime().UTC().Truncate(24 * time.Hour)
	setModTimeNormalized := modTime.UTC().Truncate(24 * time.Hour)

	t.Logf("createTempFile: Set modTime=%v (Normalized: %v), Read modTime=%v (Normalized: %v)",
		modTime, setModTimeNormalized, fileInfo.ModTime(), readModTimeNormalized)

	// Assert that the normalized read time matches the normalized set time
	assert.True(t, readModTimeNormalized.Equal(setModTimeNormalized),
		"Normalized read modification time (%v) should match normalized set modification time (%v)",
		readModTimeNormalized, setModTimeNormalized)

	return filePath
}

// Helper function to create a temporary directory with specific mod time
func createTempDir(t *testing.T, parentDir, name string, modTime time.Time) string {
	dirPath := filepath.Join(parentDir, name)
	err := os.MkdirAll(dirPath, 0755)
	assert.NoError(t, err)

	err = os.Chtimes(dirPath, modTime, modTime)
	assert.NoError(t, err)

	dirInfo, err := os.Stat(dirPath)
	assert.NoError(t, err)
	// Normalize the read time to UTC and truncate to day for consistent comparison
	readModTimeNormalized := dirInfo.ModTime().UTC().Truncate(24 * time.Hour)
	setModTimeNormalized := modTime.UTC().Truncate(24 * time.Hour)

	t.Logf("createTempDir: Set modTime=%v (Normalized: %v), Read modTime=%v (Normalized: %v)",
		modTime, setModTimeNormalized, dirInfo.ModTime(), readModTimeNormalized)

	// Assert that the normalized read time matches the normalized set time
	assert.True(t, readModTimeNormalized.Equal(setModTimeNormalized),
		"Normalized read modification time (%v) should match normalized set modification time (%v)",
		readModTimeNormalized, setModTimeNormalized)

	return dirPath
}

func TestScanProjectJunk(t *testing.T) {
	cleanupLog, err := logger.InitLogger(slog.LevelDebug)
	assert.NoError(t, err)
	defer cleanupLog()

	// Mock core.CheckGuardrails to always return false for testing purposes unless explicitly set
	oldCheckGuardrails := core.CheckGuardrails
	core.CheckGuardrails = func(path string, cfg config.Config) bool { return false }
	defer func() { core.CheckGuardrails = oldCheckGuardrails }()

	// Fixed reference time for consistent OlderThanDays calculations
	refTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("basic scan finds junk", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "scan_basic")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create a junk directory
		nmPath := filepath.Join(tempDir, "node_modules")
		err = os.Mkdir(nmPath, 0755)
		assert.NoError(t, err)

		dirModTime := refTime.Add(-5*24*time.Hour)
		createTempFile(t, nmPath, "package.js", 1024*1024, dirModTime)

		// Set directory modification time (important since we now use dir's own modtime)
		err = os.Chtimes(nmPath, dirModTime, dirModTime)
		assert.NoError(t, err)

		scanCfg := config.ScanConfig{
			Directories: []string{"node_modules"},
			Rules:       config.RuleConfig{OlderThanDays: 0},
		}
		cfg := config.Config{Scan: scanCfg}

		junkItems, err := core.ScanProjectJunk(tempDir, scanCfg, []string{}, cfg, refTime, nil)
		assert.NoError(t, err)
		assert.Len(t, junkItems, 1)
		assert.Equal(t, filepath.Clean(nmPath), filepath.Clean(junkItems[0].Path))
		assert.InDelta(t, 1.0, junkItems[0].SizeMB, 0.1)
		t.Logf("basic_scan_finds_junk: currentTime=%v, dirModTime=%v, LastModifiedDays=%d", refTime, dirModTime, junkItems[0].LastModifiedDays)
		assert.Equal(t, 5, junkItems[0].LastModifiedDays)
		assert.True(t, junkItems[0].RulesApplied)
	})

	t.Run("filters by OlderThanDays", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "scan_older_than")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create a junk directory older than 10 days
		oldNmPath := filepath.Join(tempDir, "old_node_modules")
		err = os.Mkdir(oldNmPath, 0755)
		assert.NoError(t, err)
		oldDirModTime := refTime.Add(-15*24*time.Hour)
		createTempFile(t, oldNmPath, "old_package.js", 1024*1024, oldDirModTime)
		// Set directory modification time
		err = os.Chtimes(oldNmPath, oldDirModTime, oldDirModTime)
		assert.NoError(t, err)

		// Create a junk directory newer than 10 days
		newNmPath := filepath.Join(tempDir, "new_node_modules")
		err = os.Mkdir(newNmPath, 0755)
		assert.NoError(t, err)
		newDirModTime := refTime.Add(-5*24*time.Hour)
		createTempFile(t, newNmPath, "new_package.js", 1024*1024, newDirModTime)
		// Set directory modification time
		err = os.Chtimes(newNmPath, newDirModTime, newDirModTime)
		assert.NoError(t, err)

		scanCfg := config.ScanConfig{
			Directories: []string{"old_node_modules", "new_node_modules"},
			Rules:       config.RuleConfig{OlderThanDays: 10},
		}
		cfg := config.Config{Scan: scanCfg}

		junkItems, err := core.ScanProjectJunk(tempDir, scanCfg, []string{}, cfg, refTime, nil)
		assert.NoError(t, err)
		assert.Len(t, junkItems, 2)

		// Check old_node_modules
		foundOld := false
		for _, item := range junkItems {
			if filepath.Base(item.Path) == "old_node_modules" {
				t.Logf("filters_by_OlderThanDays: old_node_modules: currentTime=%v, modTime=%v, LastModifiedDays=%d, RulesApplied=%t", refTime, oldDirModTime, item.LastModifiedDays, item.RulesApplied)
				assert.True(t, item.RulesApplied)
				foundOld = true
			}
		}
		assert.True(t, foundOld, "old_node_modules should be found and rules applied")

		// Check new_node_modules
		foundNew := false
		for _, item := range junkItems {
			if filepath.Base(item.Path) == "new_node_modules" {
				t.Logf("filters_by_OlderThanDays: new_node_modules: currentTime=%v, modTime=%v, LastModifiedDays=%d, RulesApplied=%t", refTime, newDirModTime, item.LastModifiedDays, item.RulesApplied)
				assert.False(t, item.RulesApplied)
				foundNew = true
			}
		}
		assert.True(t, foundNew, "new_node_modules should be found but rules not applied")
	})

	t.Run("respects global ignore paths", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "scan_global_ignore")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create a junk directory that should be ignored globally
		ignoredPath := filepath.Join(tempDir, "ignored_node_modules")
		err = os.Mkdir(ignoredPath, 0755)
		assert.NoError(t, err)
		createTempFile(t, ignoredPath, "package.js", 100, time.Now())

		// Create a junk directory that should be found
		foundPath := filepath.Join(tempDir, "found_node_modules")
		err = os.Mkdir(foundPath, 0755)
		assert.NoError(t, err)
		createTempFile(t, foundPath, "package.js", 100, time.Now())

		scanCfg := config.ScanConfig{
			Directories: []string{"ignored_node_modules", "found_node_modules"},
			Rules:       config.RuleConfig{OlderThanDays: 0},
		}
		cfg := config.Config{Scan: scanCfg}
		globalIgnore := []string{"ignored_node_modules"}

		junkItems, err := core.ScanProjectJunk(tempDir, scanCfg, globalIgnore, cfg, refTime, nil)
		assert.NoError(t, err)
		assert.Len(t, junkItems, 1)
		assert.Equal(t, filepath.Clean(foundPath), filepath.Clean(junkItems[0].Path))
	})

	t.Run("respects .janitorignore files", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "scan_janitor_ignore")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create a junk directory
		nmPath := filepath.Join(tempDir, "node_modules")
		err = os.Mkdir(nmPath, 0755)
		assert.NoError(t, err)
		createTempFile(t, nmPath, "package.js", 100, time.Now())

		// Create a subdirectory with a .janitorignore that ignores node_modules
		projectDir := filepath.Join(tempDir, "project")
		err = os.Mkdir(projectDir, 0755)
		assert.NoError(t, err)
		createTempFile(t, projectDir, ".janitorignore", 10, time.Now())
		err = os.WriteFile(filepath.Join(projectDir, ".janitorignore"), []byte("node_modules"), 0644)
		assert.NoError(t, err)

		ignoredNmPath := filepath.Join(projectDir, "node_modules")
		err = os.Mkdir(ignoredNmPath, 0755)
		assert.NoError(t, err)
		createTempFile(t, ignoredNmPath, "ignored_package.js", 100, time.Now())

		scanCfg := config.ScanConfig{
			Directories: []string{"node_modules"},
			Rules:       config.RuleConfig{OlderThanDays: 0},
		}
		cfg := config.Config{Scan: scanCfg}

		junkItems, err := core.ScanProjectJunk(tempDir, scanCfg, []string{}, cfg, refTime, nil)
		assert.NoError(t, err)
		assert.Len(t, junkItems, 1)
		assert.Equal(t, filepath.Clean(nmPath), filepath.Clean(junkItems[0].Path))
	})

	t.Run("respects CheckGuardrails", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "scan_guardrails")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create a junk directory that should be protected
		protectedPath := filepath.Join(tempDir, "protected_node_modules")
		err = os.Mkdir(protectedPath, 0755)
		assert.NoError(t, err)
		createTempFile(t, protectedPath, "package.js", 100, time.Now())

		// Temporarily mock CheckGuardrails to protect this specific path
		oldCheckGuardrails := core.CheckGuardrails
		core.CheckGuardrails = func(path string, cfg config.Config) bool {
			return filepath.Clean(path) == filepath.Clean(protectedPath)
		}
		defer func() { core.CheckGuardrails = oldCheckGuardrails }()

		scanCfg := config.ScanConfig{
			Directories: []string{"protected_node_modules"},
			Rules:       config.RuleConfig{OlderThanDays: 0},
		}
		cfg := config.Config{Scan: scanCfg}

		junkItems, err := core.ScanProjectJunk(tempDir, scanCfg, []string{}, cfg, refTime, nil)
		assert.NoError(t, err)
		assert.Len(t, junkItems, 0, "Should not find protected junk items")
	})
}

func TestGetDirSizeAndModTime(t *testing.T) {
	cleanupLog, err := logger.InitLogger(slog.LevelDebug)
	assert.NoError(t, err)
	defer cleanupLog()

	baseTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("calculates size and mod time for single file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "getsize_single_file")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		modTime := baseTime.Add(-24 * time.Hour)
		createTempFile(t, tempDir, "file.txt", 100, modTime)

		size, mTime, err := core.GetDirSizeAndModTime(tempDir)
		assert.NoError(t, err)
		assert.Equal(t, int64(100), size)
		assert.True(t, mTime.Equal(modTime) || mTime.After(modTime), "Modification time should be equal to or after the set time")
		assert.True(t, mTime.Before(time.Now().Add(time.Second)), "Modification time should not be in the future")
	})

	t.Run("calculates size and mod time for multiple files", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "getsize_multiple_files")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		modTime1 := baseTime.Add(-2 * 24 * time.Hour)
		modTime2 := baseTime.Add(-1 * 24 * time.Hour)

		createTempFile(t, tempDir, "file1.txt", 50, modTime1)
		createTempFile(t, tempDir, "file2.txt", 75, modTime2)

		size, mTime, err := core.GetDirSizeAndModTime(tempDir)
		assert.NoError(t, err)
		assert.Equal(t, int64(125), size)
		assert.True(t, mTime.Equal(modTime2) || mTime.After(modTime2), "Modification time should be equal to or after the latest set time")
		assert.True(t, mTime.Before(time.Now().Add(time.Second)), "Modification time should not be in the future")
	})

	t.Run("calculates size and mod time for nested directories", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "getsize_nested")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		modTimeOld := baseTime.Add(-5 * 24 * time.Hour)
		modTimeNew := baseTime.Add(-1 * 24 * time.Hour)

		createTempFile(t, tempDir, "root_file.txt", 20, modTimeOld)

		subDir1 := createTempDir(t, tempDir, "sub1", modTimeOld)
		createTempFile(t, subDir1, "sub1_file.txt", 30, modTimeOld)

		subDir2 := createTempDir(t, tempDir, "sub2", modTimeNew)
		createTempFile(t, subDir2, "sub2_file.txt", 40, modTimeNew)

		size, mTime, err := core.GetDirSizeAndModTime(tempDir)
		assert.NoError(t, err)
		assert.Equal(t, int64(90), size)
		assert.True(t, mTime.Equal(modTimeNew) || mTime.After(modTimeNew), "Modification time should be equal to or after the latest set time")
		assert.True(t, mTime.Before(time.Now().Add(time.Second)), "Modification time should not be in the future")
	})

	t.Run("handles empty directory", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "getsize_empty")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		size, mTime, err := core.GetDirSizeAndModTime(tempDir)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), size)
		// Mod time should be the directory's own mod time if no files
		dirInfo, _ := os.Stat(tempDir)
		assert.True(t, mTime.Equal(dirInfo.ModTime()) || mTime.After(dirInfo.ModTime()), "Modification time should be equal to or after the directory's mod time")
		assert.True(t, mTime.Before(time.Now().Add(time.Second)), "Modification time should not be in the future")
	})

	// Test case for permission denied error during walk
	t.Run("handles permission denied during walk", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permission denied test on Windows due to different permission model")
		}

		tempDir, err := os.MkdirTemp("", "getsize_permission_denied")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create a subdirectory with restricted permissions
		restrictedDir := filepath.Join(tempDir, "restricted")
		err = os.Mkdir(restrictedDir, 0000) // No permissions
		assert.NoError(t, err)

		// Attempt to get size of tempDir, which contains restrictedDir
		size, mTime, err := core.GetDirSizeAndModTime(tempDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
		assert.Equal(t, int64(0), size)
		assert.True(t, mTime.IsZero())

		// Restore permissions for cleanup
		_ = os.Chmod(restrictedDir, 0755)
	})
}
