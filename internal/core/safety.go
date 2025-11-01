package core

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hymkor/trash-go"
	"janitor/internal/logger"
	"janitor/internal/config"
)

var TrashThrow = trash.Throw

// SafeDelete moves the specified path to the system trash/recycle bin.
func SafeDelete(path string) error {
	logger.Log.Info("Attempting to move to trash", "path", path)
	err := TrashThrow(path)
	if err != nil {
		logger.Log.Error("Failed to move to trash", "path", path, "error", err)
		return err
	}
	logger.Log.Info("Moved to trash", "path", path)
	return nil
}

// CheckGuardrails checks if the given path is within a hardcoded system protected area or user-defined ignore paths.
var CheckGuardrails = func(path string, cfg config.Config) bool {
	// Normalize path for comparison
	normalizedPath := filepath.Clean(path)

	// Hardcoded system protected directories
	protectedPaths := []string{
		"/System",
		"/Library",
		"/Applications",
	}

	// Add Windows specific protected paths
	if runtime.GOOS == "windows" {
		protectedPaths = append(protectedPaths, "C:\\Windows", "C:\\Program Files", "C:\\Program Files (x86)")
	}

	// Add user-defined ignore paths from config
	protectedPaths = append(protectedPaths, cfg.IgnorePaths...)

	for _, p := range protectedPaths {
		normalizedProtectedPath := filepath.Clean(p)

		comparePath := normalizedPath
		compareProtectedPath := normalizedProtectedPath

		if runtime.GOOS == "windows" {
			comparePath = strings.ToLower(normalizedPath)
			compareProtectedPath = strings.ToLower(normalizedProtectedPath)
		}

		if strings.HasPrefix(comparePath, compareProtectedPath) {
			logger.Log.Warn("Path is within a protected area", "path", path, "protected_area", p)
			return true
		}
	}
	return false
}