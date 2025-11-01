package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory for config files
	tempDir, err := os.MkdirTemp("", "janitor_test_config")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test case 1: Valid config file
	validConfigContent := `
scan:
  directories:
    - node_modules
    - target
  rules:
    older_than_days: 30
ignore_paths:
  - /System
custom_cleaners:
  - name: Test Cleaner
    estimate_command: echo "100MB"
    clean_command: rm -rf /tmp/test
    requires_sudo: false
`
	validConfigPath := filepath.Join(tempDir, "config.yml")
	err = os.WriteFile(validConfigPath, []byte(validConfigContent), 0644)
	assert.NoError(t, err)

	cfg, err := LoadConfig(validConfigPath)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, 2, len(cfg.Scan.Directories))
	assert.Equal(t, "node_modules", cfg.Scan.Directories[0])
	assert.Equal(t, 30, cfg.Scan.Rules.OlderThanDays)
	assert.Equal(t, 1, len(cfg.IgnorePaths))
	assert.Equal(t, "/System", cfg.IgnorePaths[0])
	assert.Equal(t, 1, len(cfg.CustomCleaners))
	assert.Equal(t, "Test Cleaner", cfg.CustomCleaners[0].Name)

	// Test case 2: Non-existent config file (should return default config)
	nonExistentConfigPath := filepath.Join(tempDir, "non_existent.yml")
	cfg, err = LoadConfig(nonExistentConfigPath)
	assert.NoError(t, err) // Should not return error, but default config
	assert.NotNil(t, cfg)
	assert.Equal(t, DefaultConfig(), cfg) // Assert it returns the default config

	// Test case 3: Invalid YAML content
	invalidConfigContent := `
scan:
  directories:
    - node_modules
  rules:
    older_than_days: abc
`
	invalidConfigPath := filepath.Join(tempDir, "invalid_config.yml")
	err = os.WriteFile(invalidConfigPath, []byte(invalidConfigContent), 0644)
	assert.NoError(t, err)

	cfg, err = LoadConfig(invalidConfigPath)
	assert.Error(t, err) // Should return an error for invalid YAML
	assert.Equal(t, Config{}, cfg) // Should return zero-value Config on error, not nil
}

func TestGetDefaultConfigPath(t *testing.T) {
	path, err := GetDefaultConfigPath()
	assert.NoError(t, err)
	assert.True(t, strings.HasSuffix(path, filepath.Join("janitor", "config.yml")))
	assert.NotEmpty(t, path)
}