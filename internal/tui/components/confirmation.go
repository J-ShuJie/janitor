package components

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ConfirmationModel struct {
	question string
	confirmed bool
	quitting  bool
}

func NewConfirmationModel(question string) ConfirmationModel {
	return ConfirmationModel{
		question: question,
	}
}

func (m ConfirmationModel) Init() tea.Cmd {
	return nil
}

type ConfirmationMsg bool

func (m ConfirmationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.confirmed = true
			return m, func() tea.Msg { return ConfirmationMsg(true) }
		case "n", "N", "esc":
			m.confirmed = false
			return m, func() tea.Msg { return ConfirmationMsg(false) }
		}
	}
	return m, nil
}

func (m ConfirmationModel) View() string {
	buttonStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("10")).Padding(0, 1)
	selectedButtonStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("11")).Padding(0, 1)

	yesButton := buttonStyle.Render("Yes (y)")
	noButton := buttonStyle.Render("No (n)")

	if m.confirmed {
		yesButton = selectedButtonStyle.Render("Yes (y)")
	} else {
		noButton = selectedButtonStyle.Render("No (n)")
	}

	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).Render(
		fmt.Sprintf("%s\n\n%s %s", m.question, yesButton, noButton),
	)
}
