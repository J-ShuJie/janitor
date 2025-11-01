package core

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"janitor/internal/config"
	"janitor/internal/logger"
	"janitor/internal/core/ignore"
)

type JunkItem struct {
	Path           string
	SizeMB         float64
	LastModifiedDays int
	RulesApplied   bool
}

// ScanProgress represents the current progress of a scan operation
type ScanProgress struct {
	DirectoriesScanned int     // Total directories scanned so far
	JunkFoundMB        float64 // Total size of junk found (in MB)
	JunkItemsCount     int     // Number of junk items found
}

// ScanProjectJunk scans the root directory for junk items based on configuration.
// If progressChan is not nil, it will send periodic progress updates during the scan.
func ScanProjectJunk(rootDir string, scanConfig config.ScanConfig, globalIgnorePaths []string, cfg config.Config, currentTime time.Time, progressChan chan<- ScanProgress) ([]JunkItem, error) {
	logger.Log.Info("Starting scan", "rootDir", rootDir)

	// Use a context to allow cancellation if needed (e.g., from TUI)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Progress tracking with atomic counters for thread safety
	var directoriesScanned int64
	var junkItemsCount int64
	var junkTotalSizeMB int64 // Store as int64 (in KB) for atomic operations, convert to MB when reporting

	// Goroutine pool for calculating directory sizes
	pathsToProcess := make(chan string, 100) // Buffer to prevent blocking WalkDir
	results := make(chan JunkItem, 100)
	var wg sync.WaitGroup

	// Start progress reporter goroutine if channel is provided
	var progressWg sync.WaitGroup
	var progressTicker *time.Ticker
	if progressChan != nil {
		progressTicker = time.NewTicker(100 * time.Millisecond) // Send updates every 100ms
		progressWg.Add(1)
		go func() {
			defer progressWg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-progressTicker.C:
					select {
					case progressChan <- ScanProgress{
						DirectoriesScanned: int(atomic.LoadInt64(&directoriesScanned)),
						JunkFoundMB:        float64(atomic.LoadInt64(&junkTotalSizeMB)) / 1024.0, // Convert KB to MB
						JunkItemsCount:     int(atomic.LoadInt64(&junkItemsCount)),
					}:
					default:
						// Don't block if channel is full
					}
				}
			}
		}()
	}

	// Start worker goroutines
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 { // Ensure at least one worker
		numWorkers = 1
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range pathsToProcess {
				sizeMB, modTime, err := GetDirSizeAndModTime(path)
				if err != nil {
					logger.Log.Warn("Failed to get dir size or mod time", "path", path, "error", err)
					continue
				}

				// Calculate day difference based on calendar days.
				// Truncate both times to the beginning of their respective days in UTC.
				t1Day := currentTime.UTC().Truncate(24 * time.Hour)
				t2Day := modTime.UTC().Truncate(24 * time.Hour)

				// Calculate the difference in full 24-hour periods (days).
				// This gives the number of full days between the two dates.
				var lastModifiedDays int
				durationDays := int(t1Day.Sub(t2Day).Hours() / 24)

				// If the modification time is today or in the future, it's 0 days old.
				// Otherwise, it's durationDays.
				if durationDays < 0 {
					lastModifiedDays = 0
				} else {
					lastModifiedDays = durationDays
				}

				logger.Log.Debug("Calculating LastModifiedDays", "path", path, "modTime", modTime, "currentTime", currentTime, "calculatedDays", lastModifiedDays)
				rulesApplied := false
				if scanConfig.Rules.OlderThanDays == 0 || (scanConfig.Rules.OlderThanDays > 0 && lastModifiedDays >= scanConfig.Rules.OlderThanDays) {
					rulesApplied = true
				}

				itemSizeMB := float64(sizeMB) / (1024 * 1024)

				// Update progress counters
				atomic.AddInt64(&junkItemsCount, 1)
				atomic.AddInt64(&junkTotalSizeMB, int64(itemSizeMB*1024)) // Store in KB for precision

				results <- JunkItem{
					Path:           path,
					SizeMB:         itemSizeMB,
					LastModifiedDays: lastModifiedDays,
					RulesApplied:   rulesApplied,
				}
			}
		}()
	}

	// Global ignore matcher from config.IgnorePaths
	globalMatcher, err := ignore.NewIgnoreMatcherFromPatterns(globalIgnorePaths)
	if err != nil {
		logger.Log.Error("Failed to create global ignore matcher", "error", err)
		return nil, err
	}

	// Cache for .janitorignore matchers per directory
	janitorIgnoreMatchers := make(map[string]*ignore.IgnoreMatcher)
	var janitorIgnoreMatchersMu sync.RWMutex

	var foundJunk []JunkItem

	// Walk the directory tree
	walkErr := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			logger.Log.Debug("WalkDir error", "path", path, "error", err)
			// If a directory cannot be accessed, skip it but continue with others
			if os.IsPermission(err) {
				logger.Log.Warn("Permission denied, skipping directory", "path", path)
				return filepath.SkipDir
			}
			return err
		}

		// Update directory scan count
		if d.IsDir() {
			atomic.AddInt64(&directoriesScanned, 1)
		}

		// Determine relative path for ignore matching
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			logger.Log.Error("Failed to get relative path", "path", path, "rootDir", rootDir, "error", err)
			return err
		}
		if relPath == "." {
			relPath = ""
		}

		// Apply global ignore rules
		if globalMatcher.Matches(relPath) {
			if d.IsDir() {
				logger.Log.Debug("Globally ignored directory", "path", path)
				return filepath.SkipDir
			}
			return nil // Globally ignored file
		}

		// Check for .janitorignore in current directory or parent directories
		currentDir := filepath.Dir(path)
		janitorIgnoreMatchersMu.RLock()
		matcher, ok := janitorIgnoreMatchers[currentDir]
		janitorIgnoreMatchersMu.RUnlock()

		if !ok {
			// If not cached, try to load .janitorignore for this directory
			janitorIgnorePath := filepath.Join(currentDir, ".janitorignore")
			newMatcher, err := ignore.NewIgnoreMatcher(janitorIgnorePath)
			if err != nil {
				logger.Log.Warn("Failed to load .janitorignore", "path", janitorIgnorePath, "error", err)
			} else {
				janitorIgnoreMatchersMu.Lock()
				janitorIgnoreMatchers[currentDir] = newMatcher
				janitorIgnoreMatchersMu.Unlock()
				matcher = newMatcher
			}
		}

		// Apply .janitorignore rules
		if matcher != nil && matcher.Matches(filepath.Base(path)) {
			if d.IsDir() {
				logger.Log.Debug(".janitorignore ignored directory", "path", path)
				return filepath.SkipDir
			}
			return nil // .janitorignore ignored file
		}

		// Check if this directory matches any in scanConfig.Directories
		for _, targetDirName := range scanConfig.Directories {
			if d.IsDir() && d.Name() == targetDirName {
				// If it's a target directory, send it to a worker for size calculation
				// Also check guardrails here to prevent scanning protected junk directories
					if !CheckGuardrails(path, cfg) {
				
					select {
					case pathsToProcess <- path:
					case <-ctx.Done():
						return ctx.Err()
					}
					return filepath.SkipDir // Skip further traversal into this junk directory
				} else {
					logger.Log.Warn("Skipping protected junk directory from scan", "path", path)
					return filepath.SkipDir // Skip protected directory
				}
			}
		}

		return nil
	})

	close(pathsToProcess)

	// Wait for all workers to finish
	wg.Wait()
	close(results)

	// Collect all results
	for item := range results {
		foundJunk = append(foundJunk, item)
	}

	// Stop progress reporter and send final update
	if progressTicker != nil {
		progressTicker.Stop()
		cancel() // Signal progress goroutine to stop
		progressWg.Wait()

		// Send final progress update
		select {
		case progressChan <- ScanProgress{
			DirectoriesScanned: int(atomic.LoadInt64(&directoriesScanned)),
			JunkFoundMB:        float64(atomic.LoadInt64(&junkTotalSizeMB)) / 1024.0,
			JunkItemsCount:     int(atomic.LoadInt64(&junkItemsCount)),
		}:
		default:
			// Don't block if channel is full
		}
	}

	logger.Log.Info("Scan finished", "rootDir", rootDir, "junk_items_found", len(foundJunk))
	return foundJunk, walkErr
}

// getDirSizeAndModTime calculates the total size of a directory and its last modification time.
// For modification time, we use the directory's own modification time rather than its contents,
// as this better represents when the directory was last actively used (files added/removed).
func GetDirSizeAndModTime(path string) (int64, time.Time, error) {
	var totalSize int64

	// Get the directory's own modification time first
	dirInfo, err := os.Stat(path)
	if err != nil {
		return 0, time.Time{}, err
	}
	dirModTime := dirInfo.ModTime().UTC()

	// Walk the directory to calculate total size
	walkErr := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		// For files, add size
		if !info.IsDir() {
			totalSize += info.Size()
		}

		return nil
	})

	if walkErr != nil {
		return 0, time.Time{}, walkErr
	}

	return totalSize, dirModTime, nil
}