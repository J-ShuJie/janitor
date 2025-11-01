package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"

	"janitor/internal/config"
)

type ConfigModel struct {
	Config config.Config
	width  int
	height int
}

func NewConfigModel(cfg config.Config) ConfigModel {
	return ConfigModel{
		Config: cfg,
	}
}

func (m ConfigModel) Init() tea.Cmd {
	return nil
}

func (m ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg { return backToMenuMsg{} }
		}
	}
	return m, nil
}

func (m ConfigModel) View() string {
	sb := strings.Builder{}
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Current Configuration:") + "\n\n")

	// Marshal config to YAML for display
	data, err := yaml.Marshal(m.Config)
	if err != nil {
		return fmt.Sprintf("Error marshaling config: %v", err)
	}
	sb.WriteString(string(data))

	sb.WriteString("\n\n(Press Esc/Q to go back)")

	return sb.String()
}
