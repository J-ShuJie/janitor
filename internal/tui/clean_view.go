package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"janitor/internal/config"
	"janitor/internal/core"
	"janitor/internal/logger"
	"janitor/internal/tui/components"
)

type CleanerItem struct {
	Name          string
	Command       string
	EstimateSize  string // Estimated size, e.g., "1.5 GB"
	RequiresSudo  bool
	IsCustom      bool // True if from config.CustomCleaners
}

type CleanModel struct {
	Config   config.Config
	items    []CleanerItem
	cursor   int
	selected map[int]struct{}
	scanning bool // True if detecting cleaners/estimating sizes
	spinner  components.SpinnerModel
	output   []string // To display real-time output of cleaning commands
	cleaning bool     // True if a cleaning command is running
	width    int
	height   int
}

func NewCleanModel(cfg config.Config) CleanModel {
	return CleanModel{
		Config:   cfg,
		selected: make(map[int]struct{}),
		scanning: true, // Start in scanning state for detection
		spinner:  components.NewSpinnerModel("Detecting cleaners and estimating sizes..."),
	}
}

type cleanDetectionFinishedMsg struct {
	items []CleanerItem
	 err   error
}

func (m CleanModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Init(),
		func() tea.Msg {
			var detectedCleaners []CleanerItem

			// Add custom cleaners from config
			for _, cc := range m.Config.CustomCleaners {
				detectedCleaners = append(detectedCleaners, CleanerItem{
					Name:         cc.Name,
					Command:      cc.CleanCommand,
					EstimateSize: runEstimateCommand(cc.EstimateCommand),
					RequiresSudo: cc.RequiresSudo,
					IsCustom:     true,
				})
			}

			// Detect common tools and their cache sizes
			if _, err := exec.LookPath("docker"); err == nil {
				detectedCleaners = append(detectedCleaners, CleanerItem{
					Name:         "Docker System Prune",
					Command:      "docker system prune -f",
					EstimateSize: runEstimateCommand("docker system df -v | grep 'Total reclaimed space:' | awk '{print $4 $5}'"),
					RequiresSudo: false,
					IsCustom:     false,
				})
			}
			if _, err := exec.LookPath("npm"); err == nil {
				detectedCleaners = append(detectedCleaners, CleanerItem{
					Name:         "NPM Cache Clean",
					Command:      "npm cache clean --force",
					EstimateSize: runEstimateCommand("du -sh $(npm config get cache) | awk '{print $1}'"),
					RequiresSudo: false,
					IsCustom:     false,
				})
			}
			if _, err := exec.LookPath("go"); err == nil {
				detectedCleaners = append(detectedCleaners, CleanerItem{
					Name:         "Go Mod Cache Clean",
					Command:      "go clean -modcache",
					EstimateSize: runEstimateCommand("du -sh $(go env GOCACHE) | awk '{print $1}'"),
					RequiresSudo: false,
					IsCustom:     false,
				})
			}
			if _, err := exec.LookPath("pip"); err == nil {
				detectedCleaners = append(detectedCleaners, CleanerItem{
					Name:         "Pip Cache Purge",
					Command:      "pip cache purge",
					EstimateSize: runEstimateCommand("du -sh ~/.cache/pip | awk '{print $1}'"),
					RequiresSudo: false,
					IsCustom:     false,
				})
			}
			if _, err := exec.LookPath("cargo"); err == nil {
				detectedCleaners = append(detectedCleaners, CleanerItem{
					Name:         "Cargo Cache Clean",
					Command:      "cargo cache --autoclean",
					EstimateSize: runEstimateCommand("du -sh ~/.cargo/registry | awk '{print $1}'"),
					RequiresSudo: false,
					IsCustom:     false,
				})
			}


			return cleanDetectionFinishedMsg{items: detectedCleaners}
		},
	)
}

func runEstimateCommand(cmdStr string) string {
	if cmdStr == "" {
		return "N/A"
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", cmdStr)
	} else {
		cmd = exec.Command("sh", "-c", cmdStr)
	}

	out, err := cmd.Output()
	if err != nil {
		logger.Log.Warn("Failed to run estimate command", "command", cmdStr, "error", err)
		return "Error"
	}

	return strings.TrimSpace(string(out))
}

type cleanerOutputMsg struct {
	line string
}

type cleanerFinishedMsg struct {
	err error
}

func runSelectedCleaners(selected map[int]struct{}, items []CleanerItem) tea.Cmd {
	var cmds []tea.Cmd
	for idx := range selected {
		item := items[idx] // Capture loop variable
		cmd := func() tea.Msg {
			outputChan, err := core.RunCleaner(item.Command, item.RequiresSudo)
			if err != nil {
				return cleanerFinishedMsg{err: fmt.Errorf("failed to run cleaner %s: %w", item.Name, err)}
			}

			// Stream output
			for line := range outputChan {
				// Send each line as a message to update the TUI
				return cleanerOutputMsg{line: line}
			}
			return cleanerFinishedMsg{err: nil}
		}
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m CleanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	if m.cleaning {
		// If cleaning, pass messages to the spinner (or a dedicated output view)
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		// Handle output messages
		if outputMsg, ok := msg.(cleanerOutputMsg); ok {
			m.output = append(m.output, outputMsg.line)
		}
		// Handle cleaning finished message
		if finishedMsg, ok := msg.(cleanerFinishedMsg); ok {
			m.cleaning = false
			if finishedMsg.err != nil {
				return m, func() tea.Msg { return errMsg(fmt.Errorf("cleaner failed: %w", finishedMsg.err)) }
			}
			return m, func() tea.Msg { return backToMenuMsg{} }
		}
	}

	if m.scanning {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.scanning || m.cleaning {
			return m, tea.Batch(cmds...) // Ignore key presses while scanning or cleaning
		}
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case " ": // Toggle selected
			if m.cursor < len(m.items) {
				// If it requires sudo, don't allow selection in TUI
				if m.items[m.cursor].RequiresSudo {
					return m, func() tea.Msg { return errMsg(fmt.Errorf("cannot select '%s': requires sudo", m.items[m.cursor].Name)) }
				}
				if _, ok := m.selected[m.cursor]; ok {
					delete(m.selected, m.cursor)
				} else {
					m.selected[m.cursor] = struct{}{}
				}
			}
		case "enter":
			if len(m.selected) > 0 {
				// Start cleaning process
				m.cleaning = true
				m.output = []string{} // Clear previous output
				cmds = append(cmds, runSelectedCleaners(m.selected, m.items))
			}
			return m, tea.Batch(cmds...)
		case "esc", "q":
			return m, func() tea.Msg { return backToMenuMsg{} }
		}

	case cleanDetectionFinishedMsg:
		m.scanning = false
		if msg.err != nil {
			return m, func() tea.Msg { return errMsg(msg.err) }
		}
		m.items = msg.items

	case cleanerOutputMsg:
		m.output = append(m.output, msg.line)

	case cleanerFinishedMsg:
		m.cleaning = false
		if msg.err != nil {
			return m, func() tea.Msg { return errMsg(fmt.Errorf("cleaner failed: %w", msg.err)) }
		}
		return m, func() tea.Msg { return backToMenuMsg{} }
	}

	return m, tea.Batch(cmds...)
}

func (m CleanModel) View() string {
	if m.scanning {
		return m.spinner.View()
	}

	if m.cleaning {
		// Display cleaning output
		s := lipgloss.NewStyle().Bold(true).Render("Cleaning in progress...") + "\n\n"
		for _, line := range m.output {
			s += line + "\n"
		}
		return s
	}

	if len(m.items) == 0 {
		return "No cleaners found. Press Esc/Q to go back."
	}

	s := lipgloss.NewStyle().Bold(true).Render("Select cleaners to run (Space to toggle, Enter to run, Esc/Q to go back):") + "\n\n"

	for i, item := range m.items {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}

		itemStr := fmt.Sprintf("[%s] %s (%s)", checked, item.Name, item.EstimateSize)

		if item.RequiresSudo {
			itemStr += " [SUDO REQUIRED]"
			itemStr = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(itemStr)
		}

		s += fmt.Sprintf("%s %s\n", cursor, itemStr)
	}

	return s
}