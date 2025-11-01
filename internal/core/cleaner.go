package core

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"sync"

	"janitor/internal/logger"
)

var ErrRequiresSudo = errors.New("this command requires sudo privileges, please run it manually outside the TUI")

// RunCleaner executes a cleaning command and streams its output.
func RunCleaner(command string, requiresSudo bool) (<-chan string, error) {
	if requiresSudo {
		return nil, ErrRequiresSudo
	}

	logger.Log.Info("Running cleaner command", "command", command)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	outputChan := make(chan string)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			outputChan <- scanner.Text()
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			logger.Log.Error("Error reading stdout", "error", err)
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			outputChan <- "[ERROR] " + scanner.Text()
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			logger.Log.Error("Error reading stderr", "error", err)
		}
	}()

	go func() {
		wg.Wait()
		close(outputChan)
		if err := cmd.Wait(); err != nil {
			logger.Log.Error("Command finished with error", "error", err)
			// This error is not returned via the channel, but logged. The caller should check the channel for output.
		}
	}()

	return outputChan, nil
}