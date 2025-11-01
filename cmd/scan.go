package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"janitor/internal/core"
	"janitor/internal/logger"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Scan for project junk (node_modules, target, etc.)",
	Long: `Scan the current directory or a specified path for common project junk
like node_modules, target directories, build artifacts, and more.`, 
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rootDir := "."
		if len(args) > 0 {
			rootDir = args[0]
		}

		deleteFlag, _ := cmd.Flags().GetBool("delete")
		forceFlag, _ := cmd.Flags().GetBool("force")
		dryRunFlag, _ := cmd.Flags().GetBool("dry-run")

		logger.Log.Info("Scan command initiated", "rootDir", rootDir, "delete", deleteFlag, "force", forceFlag, "dryRun", dryRunFlag)

		// Create progress channel and start a goroutine to print progress
		progressChan := make(chan core.ScanProgress, 10)
		done := make(chan bool)
		go func() {
			lastPrint := time.Now()
			for progress := range progressChan {
				// Print progress updates every second to avoid flooding the console
				if time.Since(lastPrint) >= time.Second {
					fmt.Printf("\rScanning... %d directories scanned, %.1f MB junk found (%d items)",
						progress.DirectoriesScanned, progress.JunkFoundMB, progress.JunkItemsCount)
					lastPrint = time.Now()
				}
			}
			done <- true
		}()

		junkItems, err := core.ScanProjectJunk(rootDir, cfg.Scan, cfg.IgnorePaths, cfg, time.Now(), progressChan)
		close(progressChan)
		<-done // Wait for progress printer to finish
		fmt.Println() // New line after progress

		if err != nil {
			logger.Log.Error("Error during scan", "error", err)
			fmt.Fprintf(os.Stderr, "Error during scan: %v\n", err)
			os.Exit(1)
		}

		if len(junkItems) == 0 {
			fmt.Println("No junk items found.")
			return
		}

		fmt.Println("Found junk items:")
		for _, item := range junkItems {
			status := ""
			if !item.RulesApplied {
				status = " [RULE SKIPPED]"
			}
			fmt.Printf("- %s (%.1f MB) (Last Modified %d days)%s\n", item.Path, item.SizeMB, item.LastModifiedDays, status)
		}

		if deleteFlag {
			fmt.Println("\nAttempting to delete selected items...")
			for _, item := range junkItems {
				if !item.RulesApplied {
					logger.Log.Info("Skipping deletion of item due to rules", "path", item.Path)
					continue
				}

				if core.CheckGuardrails(item.Path, cfg) {
					logger.Log.Warn("Skipping deletion of protected item", "path", item.Path)
					fmt.Printf("Skipping protected item: %s\n", item.Path)
					continue
				}

				if dryRunFlag {
					fmt.Printf("[DRY RUN] Would delete: %s\n", item.Path)
					continue
				}

				if forceFlag {
					logger.Log.Warn("Force deleting item (skipping trash)", "path", item.Path)
					err := os.RemoveAll(item.Path)
					if err != nil {
						logger.Log.Error("Failed to force delete item", "path", item.Path, "error", err)
						fmt.Fprintf(os.Stderr, "Error force deleting %s: %v\n", item.Path, err)
					} else {
						fmt.Printf("Force deleted: %s\n", item.Path)
					}
				} else {
					err := core.SafeDelete(item.Path)
					if err != nil {
						logger.Log.Error("Failed to move item to trash", "path", item.Path, "error", err)
						fmt.Fprintf(os.Stderr, "Error moving %s to trash: %v\n", item.Path, err)
					} else {
						fmt.Printf("Moved to trash: %s\n", item.Path)
					}
				}
			}
		} else if !dryRunFlag {
			fmt.Println("\nUse --delete to remove items, --dry-run to see what would be deleted.")
		}
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)

	// Here you can define flags and configuration settings.
	scanCmd.Flags().BoolP("delete", "d", false, "Delete found junk items")
	scanCmd.Flags().BoolP("force", "f", false, "Force permanent deletion (skip trash)")
	scanCmd.Flags().BoolP("dry-run", "n", false, "Perform a dry run without actual deletion")

	// Cobra supports local flags which only run when this command
	// is called directly, e.g.:
	// scanCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}