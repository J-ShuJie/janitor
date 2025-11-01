package tui

import (
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"janitor/internal/config"
)

func TestNewConfigModel(t *testing.T) {
	cfg := config.DefaultConfig()
	model := NewConfigModel(cfg)

	assert.Equal(t, cfg, model.Config, "NewConfigModel should initialize with the provided config")
}

func TestConfigModelUpdate(t *testing.T) {
	cfg := config.DefaultConfig()
	model := NewConfigModel(cfg)

	// Test 'esc' key to go back
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.IsType(t, ConfigModel{}, updatedModel, "Update should return a ConfigModel")
	assert.NotNil(t, cmd, "Update should return a command")

	msg := cmd()
	assert.IsType(t, backToMenuMsg{}, msg, "Command should return backToMenuMsg")

	// Test 'q' key to go back
	updatedModel, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.IsType(t, ConfigModel{}, updatedModel, "Update should return a ConfigModel")
	assert.NotNil(t, cmd, "Update should return a command")

	msg = cmd()
	assert.IsType(t, backToMenuMsg{}, msg, "Command should return backToMenuMsg")

	// Test other key press (should do nothing)
	updatedModel, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.IsType(t, ConfigModel{}, updatedModel, "Update should return a ConfigModel")
	assert.Nil(t, cmd, "Command should be nil for unhandled key")
}

func TestConfigModelView(t *testing.T) {
	cfg := config.DefaultConfig()
	model := NewConfigModel(cfg)

	view := model.View()

	// Check for header
	assert.Contains(t, view, "Current Configuration:", "View should contain the header")

	// Check for YAML content
	expectedYAML, err := yaml.Marshal(cfg)
	assert.NoError(t, err)
	assert.Contains(t, view, string(expectedYAML), "View should contain the YAML representation of the config")

	// Check for back instruction
	assert.Contains(t, view, "(Press Esc/Q to go back)", "View should contain back instruction")
}
