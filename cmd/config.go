package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"janitor/internal/config"
	"janitor/internal/logger"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Janitor configuration",
	Long:  `Allows you to edit or initialize the Janitor configuration file.`, 
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit the Janitor configuration file",
	Long:  `Opens the Janitor configuration file in your default editor.`, 
	Run: func(cmd *cobra.Command, args []string) {
		configPath, err := config.GetDefaultConfigPath()
		if err != nil {
			logger.Log.Error("Error getting default config path", "error", err)
			fmt.Fprintf(os.Stderr, "Error getting default config path: %v\n", err)
			os.Exit(1)
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			switch runtime.GOOS {
			case "windows":
				editor = "notepad.exe"
			case "darwin":
				editor = "open"
			default:
				editor = "vi"
			}
		}

		fmt.Printf("Opening configuration file %s with %s...\n", configPath, editor)
		cmdExec := exec.Command(editor, configPath)
		cmdExec.Stdin = os.Stdin
		cmdExec.Stdout = os.Stdout
		cmdExec.Stderr = os.Stderr

		if err := cmdExec.Run(); err != nil {
			logger.Log.Error("Error opening editor", "editor", editor, "path", configPath, "error", err)
			fmt.Fprintf(os.Stderr, "Error opening editor %s for %s: %v\n", editor, configPath, err)
			os.Exit(1)
		}
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a default Janitor configuration file",
	Long:  `Generates a default config.yml file in the Janitor configuration directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		configPath, err := config.GetDefaultConfigPath()
		if err != nil {
			logger.Log.Error("Error getting default config path", "error", err)
			fmt.Fprintf(os.Stderr, "Error getting default config path: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Attempting to use config path: %s\n", configPath)

		// Check if the file already exists
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Configuration file already exists at %s. Overwrite? (y/N) ", configPath)
			var response string
			fmt.Scanln(&response)

			if strings.ToLower(strings.TrimSpace(response)) != "y" {
				fmt.Println("Aborted.")
				return
			}
		}

		// Ensure the directory exists
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			logger.Log.Error("Error creating config directory", "path", configDir, "error", err)
			fmt.Fprintf(os.Stderr, "Error creating config directory %s: %v\n", configDir, err)
			os.Exit(1)
		}

		defaultConfigContent := `# Feature B: Project junk directories to scan
scan:
  directories:
    - node_modules
    - target
    - build
    - dist
    - venv
    - __pycache__
    - .gradle
    - DerivedData

  # Feature E: Advanced rules
  rules:
    # 0 means clean all projects regardless of age
    older_than_days: 30

# Feature G: Global ignore paths (protected zones)
ignore_paths:
  - /System
  - /Library
  - /Applications
  - C:\Windows
  - ~Backups

# Feature F: Custom cleaners
custom_cleaners:
  - name: Apt Cache (Debian/Ubuntu)
    # Command to estimate size for TUI list
    estimate_command: du -sh /var/cache/apt
    # Actual clean command
    clean_command: apt-get clean
    # If true, TUI will prompt user to run this command manually
    requires_sudo: true

  - name: Homebrew Cache (macOS)
    estimate_command: du -sh $(brew --cache)
    clean_command: brew cleanup
    requires_sudo: false
`
		err = os.WriteFile(configPath, []byte(defaultConfigContent), 0644)
		if err != nil {
			logger.Log.Error("Error writing default config file", "path", configPath, "error", err)
			fmt.Fprintf(os.Stderr, "Error writing default config file %s: %v\n", configPath, err)
			os.Exit(1)
		}

		fmt.Printf("Default configuration file created at %s\n", configPath)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configInitCmd)

	// Here you can define flags and configuration settings.

	// Cobra supports local flags which only run when this command
	// is called directly, e.g.:
	// configCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}