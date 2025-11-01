package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
)

type MenuItem int

const (
	ScanProjectJunkMenuItem MenuItem = iota
	CleanGlobalCachesMenuItem
	EditConfigurationMenuItem
	QuitMenuItem
)

type MenuModel struct {
	cursor int
}

func NewMenuModel() MenuModel {
	return MenuModel{}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

type selectMenuItemMsg struct {
	item MenuItem
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < 3 {
				m.cursor++
			}
		case "enter":
			return m, func() tea.Msg { return selectMenuItemMsg{MenuItem(m.cursor)} }
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	s := "🧹 Welcome to Janitor!\n\n"
	s += "What would you like to clean (Use ↑↓, Enter to select)\n\n"

	choices := []string{
		"Scan for Project Junk (node_modules, target, etc.)",
		"Clean Global Package Caches (npm, pip, cargo...)",
		"Edit Configuration",
		"Quit",
	}

	for i, choice := range choices {
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	s += "\n(Q to Quit)"
	return s
}