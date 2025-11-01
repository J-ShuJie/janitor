package core_test

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"janitor/internal/config"
	"janitor/internal/core"
	"janitor/internal/logger"
)

// MockTrash is a mock implementation of the trash-go library for testing.
type MockTrash struct {
	mock.Mock
}

func (m *MockTrash) Throw(filenames ...string) error {
	args := m.Called(filenames)
	return args.Error(0)
}

// Mock os.RemoveAll for testing force deletion
var osRemoveAll = os.RemoveAll

func TestSafeDelete(t *testing.T) {
	// Initialize logger (needed for SafeDelete logging)
	cleanup, err := logger.InitLogger(0) // Use default log level
	assert.NoError(t, err)
	defer cleanup()

	// Create a mock for the trash-go library
	mockTrash := new(MockTrash)

	// Replace the global trash.Throw with our mock for this test
	oldTrashThrow := core.TrashThrow
	core.TrashThrow = mockTrash.Throw
	defer func() { core.TrashThrow = oldTrashThrow }()

	// Test case 1: Successful deletion
	testPath := "/tmp/test_file_to_delete"
	mockTrash.On("Throw", []string{testPath}).Return(nil).Once()

	err = core.SafeDelete(testPath)
	assert.NoError(t, err)
	mockTrash.AssertExpectations(t)

	// Test case 2: Deletion fails
	mockTrash.On("Throw", []string{testPath}).Return(assert.AnError).Once()

	err = core.SafeDelete(testPath)
	assert.Error(t, err)
	mockTrash.AssertExpectations(t)
}

func TestCheckGuardrails(t *testing.T) {
	// Initialize logger (needed for CheckGuardrails logging)
	cleanup, err := logger.InitLogger(0) // Use default log level
	assert.NoError(t, err)
	defer cleanup()

	// Create a mock config with ignore paths
	mockConfig := config.Config{
		IgnorePaths: []string{
			"/home/user/protected_project",
			"C:\\Users\\Admin\\SensitiveData",
			"/var/log",
		},
	}

	// Test case 1: Protected path (Linux/macOS) with hardcoded paths
	if runtime.GOOS != "windows" {
		assert.True(t, core.CheckGuardrails("/System/Library", mockConfig))
		assert.True(t, core.CheckGuardrails("/Applications/MyApp.app", mockConfig))
		assert.True(t, core.CheckGuardrails("/System", mockConfig))
		assert.True(t, core.CheckGuardrails("/Library", mockConfig))
		assert.True(t, core.CheckGuardrails("/Applications", mockConfig))
		assert.True(t, core.CheckGuardrails("/System/foo/bar", mockConfig))
		assert.True(t, core.CheckGuardrails("/Library/foo/bar", mockConfig))
		assert.True(t, core.CheckGuardrails("/Applications/foo/bar", mockConfig))
	}

	// Test case 2: Protected path (Windows) with hardcoded paths
	if runtime.GOOS == "windows" {
		assert.True(t, core.CheckGuardrails("C:\\Windows\\System32", mockConfig))
		assert.True(t, core.CheckGuardrails("C:\\Program Files\\MyApp", mockConfig))
		assert.True(t, core.CheckGuardrails("C:\\Program Files (x86)\\MyApp", mockConfig))
		assert.True(t, core.CheckGuardrails("C:\\Windows", mockConfig))
		assert.True(t, core.CheckGuardrails("C:\\Program Files", mockConfig))
		assert.True(t, core.CheckGuardrails("C:\\Program Files (x86)", mockConfig))
		assert.True(t, core.CheckGuardrails("C:\\Windows\\foo\\bar", mockConfig))
		assert.True(t, core.CheckGuardrails("C:\\Program Files\\foo\\bar", mockConfig))
		assert.True(t, core.CheckGuardrails("C:\\Program Files (x86)\\foo\\bar", mockConfig))
	}

	// Test case 3: Protected path from config.IgnorePaths
	assert.True(t, core.CheckGuardrails("/home/user/protected_project/subfolder", mockConfig))
	assert.True(t, core.CheckGuardrails("C:\\Users\\Admin\\SensitiveData\\temp.txt", mockConfig))
	assert.True(t, core.CheckGuardrails("/var/log/syslog", mockConfig))

	// Test case 4: Unprotected path
	assert.False(t, core.CheckGuardrails("/home/user/project", mockConfig))
	assert.False(t, core.CheckGuardrails("C:\\Users\\user\\Documents", mockConfig))
	assert.False(t, core.CheckGuardrails("/tmp/foo", mockConfig))
	assert.False(t, core.CheckGuardrails("D:\\Projects\\MyProject", mockConfig))

	// Test case 5: Case insensitivity (Windows) for config paths
	if runtime.GOOS == "windows" {
		assert.True(t, core.CheckGuardrails("c:\\users\\admin\\sensitivedata", mockConfig))
		assert.True(t, core.CheckGuardrails("C:\\USERS\\ADMIN\\SENSITIVEDATA\\TEMP.TXT", mockConfig))
	}

	// Test case 6: Path with different separators for config paths
	if runtime.GOOS == "windows" {
		assert.True(t, core.CheckGuardrails("C:/Users/Admin/SensitiveData/temp.txt", mockConfig))
	}
}