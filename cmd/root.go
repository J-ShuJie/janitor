package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"janitor/internal/config"
	"janitor/internal/logger"
	"janitor/internal/tui"
)

var cfg config.Config
var cleanupLog func()

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "janitor",
	Short: "Janitor is a CLI tool for cleaning up development environments",
	Long: `A fast and flexible CLI tool to clean up your development environment.
It helps you remove unwanted files and directories like node_modules,
target directories, and clear global caches.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logger first
		var err error
		cleanupLog, err = logger.InitLogger(slog.LevelDebug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing logger: %v\n", err)
			os.Exit(1)
		}

		// Load configuration
		configPath, err := config.GetDefaultConfigPath()
		if err != nil {
			logger.Log.Error("Error getting default config path", "error", err)
			return fmt.Errorf("error getting default config path: %w", err)
		}

		cfg, err = config.LoadConfig(configPath)
		if err != nil {
			logger.Log.Warn("No config file found or error loading config, using default values", "path", configPath, "error", err)
			// Optionally, initialize with default config if file not found
			// cfg = config.DefaultConfig()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		defer cleanupLog()
		// If no subcommands are given, start the TUI
		if len(args) == 0 && cmd.Flags().ArgsLenAtDash() == 0 {
			initialModel := tui.InitialModel(cfg)
			p := tea.NewProgram(initialModel, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				logger.Log.Error("Error running TUI", "error", err)
				fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
				os.Exit(1)
			}
		} else {
			cmd.Help()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you can define global flags and configuration that apply to all subcommands.
	// For example:
	// rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.janitor.yaml)")
	// rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")
}