package tui

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"janitor/internal/config"
	"janitor/internal/logger"
)

type ViewState int

const (
	MenuView ViewState = iota
	ScanView
	CleanView
	ConfigView
)

type errMsg error

type MainModel struct {
	Config      config.Config
	Logger      *slog.Logger
	currentView ViewState
	error       error
	errorTimer    *time.Timer
	menu        MenuModel
	scan        ScanModel
	clean       CleanModel
	configEditor ConfigModel // New: ConfigModel for editing configuration
	width       int
	height      int
}

func InitialModel(cfg config.Config) MainModel {
	return MainModel{
		Config:      cfg,
		Logger:      logger.Log,
		currentView: MenuView,
		menu:        NewMenuModel(),
		scan:        NewScanModel(cfg),
		clean:       NewCleanModel(cfg),
		configEditor: NewConfigModel(cfg), // New: Initialize ConfigModel
	}
}

func (m MainModel) Init() tea.Cmd {
	return tea.Batch(
		m.menu.Init(),
		m.scan.Init(),
		m.clean.Init(),
		m.configEditor.Init(), // New: Initialize ConfigModel
	)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case errMsg:
		m.error = msg
		if m.errorTimer != nil {
			m.errorTimer.Stop()
		}
		m.errorTimer = time.NewTimer(5 * time.Second)
		cmds = append(cmds, tea.Tick(5*time.Second, func(t time.Time) tea.Msg { return clearErrorMsg{} }))

	case clearErrorMsg:
		m.error = nil
		m.errorTimer = nil

	case selectMenuItemMsg:
		switch msg.item {
		case ScanProjectJunkMenuItem:
			m.currentView = ScanView
			cmds = append(cmds, m.scan.Init())
		case CleanGlobalCachesMenuItem:
			m.currentView = CleanView
			cmds = append(cmds, m.clean.Init())
		case EditConfigurationMenuItem:
			m.currentView = ConfigView
			cmds = append(cmds, m.configEditor.Init()) // New: Initialize ConfigModel
		case QuitMenuItem:
			return m, tea.Quit
		}
	case backToMenuMsg:
		m.currentView = MenuView
	}

	// Update the current view
	switch m.currentView {
	case MenuView:
		menuModel, menuCmd := m.menu.Update(msg)
		m.menu = menuModel.(MenuModel)
		cmds = append(cmds, menuCmd)
	case ScanView:
		scanModel, scanCmd := m.scan.Update(msg)
		m.scan = scanModel.(ScanModel)
		cmds = append(cmds, scanCmd)
	case CleanView:
		cleanModel, cleanCmd := m.clean.Update(msg)
		m.clean = cleanModel.(CleanModel)
		cmds = append(cmds, cleanCmd)
	case ConfigView:
		configModel, configCmd := m.configEditor.Update(msg)
		m.configEditor = configModel.(ConfigModel)
		cmds = append(cmds, configCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) View() string {
	var s string

	switch m.currentView {
	case MenuView:
		s = m.menu.View()
	case ScanView:
		s = m.scan.View()
	case CleanView:
		s = m.clean.View()
	case ConfigView:
		s = m.configEditor.View() // New: Render ConfigModel's view
	}

	if m.error != nil {
		errorStyle := lipgloss.NewStyle().Background(lipgloss.Color("9")).Foreground(lipgloss.Color("0")).Padding(0, 1).Width(m.width)
		s += "\n" + errorStyle.Render(fmt.Sprintf("[ERROR] %v", m.error))
	}

	return s
}

type clearErrorMsg struct{}
type backToMenuMsg struct{}