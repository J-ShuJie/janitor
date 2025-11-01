package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"janitor/internal/config"
	"janitor/internal/core"
	"janitor/internal/logger"
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "cleancache",
	Short: "Clean global package caches (npm, pip, cargo...)",
	Long: `Clean various global package and system caches to free up disk space.
This command can detect common package managers and offer to clean their caches.`, 
	Run: func(cmd *cobra.Command, args []string) {
		allFlag, _ := cmd.Flags().GetBool("all")
		forceFlag, _ := cmd.Flags().GetBool("force")

		logger.Log.Info("Clean cache command initiated", "all", allFlag, "force", forceFlag)

		// Define a list of potential cleaners
		var cleaners []config.CustomCleaner

		// Add detected system cleaners
		if _, err := exec.LookPath("docker"); err == nil {
			cleaners = append(cleaners, config.CustomCleaner{Name: "Docker System Prune", CleanCommand: "docker system prune -f", RequiresSudo: false})
		}
		if _, err := exec.LookPath("npm"); err == nil {
			cleaners = append(cleaners, config.CustomCleaner{Name: "NPM Cache Clean", CleanCommand: "npm cache clean --force", RequiresSudo: false})
		}
		if _, err := exec.LookPath("pip"); err == nil {
			cleaners = append(cleaners, config.CustomCleaner{Name: "Pip Cache Purge", CleanCommand: "pip cache purge", RequiresSudo: false})
		}
		if _, err := exec.LookPath("cargo"); err == nil {
			cleaners = append(cleaners, config.CustomCleaner{Name: "Cargo Cache Clean", CleanCommand: "cargo cache --autoclean", RequiresSudo: false})
		}
		// Add custom cleaners from config
		cleaners = append(cleaners, cfg.CustomCleaners...)

		if len(cleaners) == 0 {
			fmt.Println("No cleaners found or configured.")
			return
		}

		fmt.Println("Available Cleaners:")
		for i, cleaner := range cleaners {
			sudo := ""
			if cleaner.RequiresSudo {
				sudo = " [SUDO REQUIRED]"
			}
			fmt.Printf("  %d. %s%s\n", i+1, cleaner.Name, sudo)
		}

		if allFlag {
			fmt.Println("\nRunning all available cleaners (excluding those requiring sudo)...")
			for _, cleaner := range cleaners {
				if cleaner.RequiresSudo {
					logger.Log.Info("Skipping cleaner requiring sudo", "name", cleaner.Name)
					fmt.Printf("Skipping %s (requires sudo).\n", cleaner.Name)
					continue
				}
				fmt.Printf("\n--- Running %s ---\n", cleaner.Name)
				outputChan, err := core.RunCleaner(cleaner.CleanCommand, cleaner.RequiresSudo)
				if err != nil {
					logger.Log.Error("Error running cleaner", "name", cleaner.Name, "error", err)
					fmt.Fprintf(os.Stderr, "Error running %s: %v\n", cleaner.Name, err)
					continue
				}
				for line := range outputChan {
					fmt.Println(line)
				}
				fmt.Printf("--- Finished %s ---\n", cleaner.Name)
			}
		} else {
			fmt.Println("\nUse --all to run all available cleaners (excluding those requiring sudo).")
			fmt.Println("For cleaners requiring sudo, please run them manually.")
		}
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)

	// Here you can define flags and configuration settings.
	cleanCmd.Flags().BoolP("all", "a", false, "Run all available cleaners (excluding those requiring sudo)")
	cleanCmd.Flags().BoolP("force", "f", false, "Force cleaning operations (use with caution)")

	// Cobra supports local flags which only run when this command
	// is called directly, e.g.:
	// cleanCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}