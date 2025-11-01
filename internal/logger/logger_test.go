package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper to capture stderr output for testing
func captureStderr(f func()) string {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stderr = oldStderr
	return string(out)
}

func TestInitLogger(t *testing.T) {
	t.Run("successful initialization and logging", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "janitor_log_test")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Mock os.UserCacheDir to point to our temp directory
		oldUserCacheDir := osUserCacheDir
		osUserCacheDir = func() (string, error) { return tempDir, nil }
		defer func() { osUserCacheDir = oldUserCacheDir }()

		cleanup, err := InitLogger(slog.LevelDebug)
		assert.NoError(t, err)
		assert.NotNil(t, cleanup)
		defer cleanup()

		Log.Info("test message", "key", "value")

		logFilePath := filepath.Join(tempDir, "janitor", "janitor.log")
		assert.FileExists(t, logFilePath)

		content, err := os.ReadFile(logFilePath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "\"level\":\"INFO\"", "Log level should be INFO")
		assert.Contains(t, string(content), "\"msg\":\"test message\"", "Log message should be present")
		assert.Contains(t, string(content), "\"key\":\"value\"", "Log attributes should be present")
	})

	t.Run("error getting cache dir", func(t *testing.T) {
		// Mock os.UserCacheDir to return an error
		oldUserCacheDir := osUserCacheDir
		osUserCacheDir = func() (string, error) { return "", fmt.Errorf("mock error") }
		defer func() { osUserCacheDir = oldUserCacheDir }()

		cleanup, err := InitLogger(slog.LevelDebug)
		assert.Error(t, err)
		assert.Nil(t, cleanup)
		assert.Contains(t, err.Error(), "failed to get user cache directory")
	})

	t.Run("error creating log dir", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "janitor_log_test")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Mock os.UserCacheDir to point to our temp directory
		oldUserCacheDir := osUserCacheDir
		osUserCacheDir = func() (string, error) { return tempDir, nil }
		defer func() { osUserCacheDir = oldUserCacheDir }()

		// Create a file with the same name as the intended log directory to cause MkdirAll to fail
		_ = os.WriteFile(filepath.Join(tempDir, "janitor"), []byte("not a dir"), 0644)

		cleanup, err := InitLogger(slog.LevelDebug)
		assert.Error(t, err)
		assert.Nil(t, cleanup)
		assert.Contains(t, err.Error(), "failed to create log directory")
	})
}

func TestTempFileCreation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "temp_file_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "test.log")
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	assert.NotNil(t, file)
	defer file.Close()

	_, err = file.WriteString("Hello, world!")
	assert.NoError(t, err)

	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, world!", string(content))
}
