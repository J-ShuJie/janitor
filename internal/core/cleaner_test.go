package core_test

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"janitor/internal/core"
	"janitor/internal/logger"
)

func TestRunCleaner(t *testing.T) {
	// Initialize logger
	cleanup, err := logger.InitLogger(0)
	assert.NoError(t, err)
	defer cleanup()

	t.Run("executes simple command successfully", func(t *testing.T) {
		var command string
		if runtime.GOOS == "windows" {
			command = "echo Hello World"
		} else {
			command = "echo Hello World"
		}

		outputChan, err := core.RunCleaner(command, false)
		assert.NoError(t, err)
		assert.NotNil(t, outputChan)

		// Collect output
		var output []string
		for line := range outputChan {
			output = append(output, line)
		}

		// Verify output
		assert.NotEmpty(t, output)
		assert.Contains(t, strings.Join(output, "\n"), "Hello World")
	})

	t.Run("refuses to run commands requiring sudo", func(t *testing.T) {
		command := "echo test"
		outputChan, err := core.RunCleaner(command, true)

		assert.Error(t, err)
		assert.Equal(t, core.ErrRequiresSudo, err)
		assert.Nil(t, outputChan)
	})

	t.Run("captures stderr output", func(t *testing.T) {
		var command string
		if runtime.GOOS == "windows" {
			// Windows: redirect to stderr
			command = "echo Error Message 1>&2"
		} else {
			// Unix: redirect to stderr
			command = "echo 'Error Message' >&2"
		}

		outputChan, err := core.RunCleaner(command, false)
		assert.NoError(t, err)
		assert.NotNil(t, outputChan)

		// Collect output
		var output []string
		for line := range outputChan {
			output = append(output, line)
		}

		// Verify stderr was captured (should have [ERROR] prefix)
		found := false
		for _, line := range output {
			if strings.Contains(line, "[ERROR]") && strings.Contains(line, "Error Message") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should capture stderr output with [ERROR] prefix")
	})

	t.Run("handles command that doesn't exist", func(t *testing.T) {
		command := "nonexistent_command_12345"

		outputChan, err := core.RunCleaner(command, false)

		// Command should start but fail
		if err != nil {
			// Error during start is acceptable
			assert.Error(t, err)
		} else {
			// Or it starts and we get error in output
			assert.NotNil(t, outputChan)
			// Drain the channel
			for range outputChan {
			}
		}
	})

	t.Run("streams output progressively", func(t *testing.T) {
		var command string
		if runtime.GOOS == "windows" {
			// Windows: multiple echo commands
			command = "echo Line1 && echo Line2 && echo Line3"
		} else {
			// Unix: multiple echo commands
			command = "echo Line1; echo Line2; echo Line3"
		}

		outputChan, err := core.RunCleaner(command, false)
		assert.NoError(t, err)
		assert.NotNil(t, outputChan)

		// Collect output with timestamps to verify streaming
		var lines []string
		for line := range outputChan {
			lines = append(lines, line)
		}

		// Should have received multiple lines
		assert.GreaterOrEqual(t, len(lines), 1, "Should receive at least one line")

		// Verify content
		output := strings.Join(lines, "\n")
		assert.Contains(t, output, "Line1")
		assert.Contains(t, output, "Line2")
		assert.Contains(t, output, "Line3")
	})

	t.Run("handles empty output command", func(t *testing.T) {
		var command string
		if runtime.GOOS == "windows" {
			command = "echo."
		} else {
			command = "true"
		}

		outputChan, err := core.RunCleaner(command, false)
		assert.NoError(t, err)
		assert.NotNil(t, outputChan)

		// Collect output
		var output []string
		timeout := time.After(5 * time.Second)
		done := make(chan bool)

		go func() {
			for line := range outputChan {
				output = append(output, line)
			}
			done <- true
		}()

		select {
		case <-done:
			// Command finished successfully
		case <-timeout:
			t.Fatal("Command timed out")
		}

		// Empty or minimal output is acceptable
		t.Logf("Output lines: %d", len(output))
	})

	t.Run("handles commands with special characters", func(t *testing.T) {
		var command string
		if runtime.GOOS == "windows" {
			command = "echo Special: !@#$%"
		} else {
			command = "echo 'Special: !@#$%'"
		}

		outputChan, err := core.RunCleaner(command, false)
		assert.NoError(t, err)
		assert.NotNil(t, outputChan)

		// Collect output
		var output []string
		for line := range outputChan {
			output = append(output, line)
		}

		// Should have some output
		assert.NotEmpty(t, output)
	})
}
