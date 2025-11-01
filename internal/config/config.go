package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Scan          ScanConfig      `yaml:"scan"`
	IgnorePaths   []string        `yaml:"ignore_paths"`
	CustomCleaners []CustomCleaner `yaml:"custom_cleaners"`
}

type ScanConfig struct {
	Directories []string `yaml:"directories"`
	Rules       RuleConfig `yaml:"rules"`
}

type RuleConfig struct {
	OlderThanDays int `yaml:"older_than_days"`
}

type CustomCleaner struct {
	Name           string `yaml:"name"`
	EstimateCommand string `yaml:"estimate_command"`
	CleanCommand    string `yaml:"clean_command"`
	RequiresSudo    bool   `yaml:"requires_sudo"`
}

func DefaultConfig() Config {
	return Config{
		Scan: ScanConfig{
			Directories: []string{
				"node_modules",
				"target",
				"build",
				"dist",
				"venv",
				"__pycache__",
				".gradle",
				"DerivedData",
			},
			Rules: RuleConfig{
				OlderThanDays: 30,
			},
		},
		IgnorePaths: []string{
			"/System",
			"/Library",
			"/Applications",
			"C:\\Windows",
			"~Backups",
		},
		CustomCleaners: []CustomCleaner{
			{
				Name:           "Apt Cache (Debian/Ubuntu)",
				EstimateCommand: "du -sh /var/cache/apt",
				CleanCommand:    "apt-get clean",
				RequiresSudo:    true,
			},
			{
				Name:           "Homebrew Cache (macOS)",
				EstimateCommand: "du -sh $(brew --cache)",
				CleanCommand:    "brew cleanup",
				RequiresSudo:    false,
			},
		},
	}
}

func LoadConfig(path string) (Config, error) {
	var cfg Config

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil // Return default config if file not found
		}
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	return cfg, nil
}

// GetDefaultConfigPath returns the default path for the janitor config file.
func GetDefaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(configDir, "janitor", "config.yml"), nil
}