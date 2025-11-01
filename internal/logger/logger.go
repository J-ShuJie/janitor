package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

var Log *slog.Logger

// osUserCacheDir is a variable that wraps os.UserCacheDir to allow mocking in tests.
var osUserCacheDir = os.UserCacheDir

func InitLogger(level slog.Level) (func(), error) {
	cacheDir, err := osUserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user cache directory: %w", err)
	}
	logDir := filepath.Join(cacheDir, "janitor")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logFile, err := os.OpenFile(filepath.Join(logDir, "janitor.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	handler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level: level,
	})

	Log = slog.New(handler)
	slog.SetDefault(Log)

	return func() {
		_ = logFile.Close() // Ignore error on close for simplicity in this context
	}, nil
}