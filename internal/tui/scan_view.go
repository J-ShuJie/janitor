package tui

import (

	"fmt"

	"os"
	"sort"
	"time"



	"github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"



	"janitor/internal/config"

	"janitor/internal/core"

	"janitor/internal/tui/components"

)



type sortMode int

const (
	sortByNone sortMode = iota
	sortBySize
	sortByDate
)

type ScanModel struct {



	Config   config.Config



	items    []core.JunkItem



	cursor   int



	selected map[int]struct{}



	scanning bool // New field to indicate if scanning is in progress



	spinner  components.SpinnerModel // Add spinner model







	confirmation     components.ConfirmationModel // New field for confirmation dialog



	showConfirmation bool                         // New field to control confirmation visibility



	currentSortMode  sortMode                     // Current sort mode

	// Progress tracking
	directoriesScanned int
	junkFoundMB        float64
	junkItemsCount     int

	// Channels for async communication
	progressChan <-chan core.ScanProgress
	doneChan     <-chan scanFinishedMsg



	width            int



	height           int



}



func NewScanModel(cfg config.Config) ScanModel {

	return ScanModel{

		Config:   cfg,

		selected: make(map[int]struct{}),

		scanning: true, // Start in scanning state

		spinner:  components.NewSpinnerModel("Scanning for junk..."), // Initialize spinner

		confirmation: components.NewConfirmationModel("Are you sure you want to delete the selected items?"), // Initialize confirmation

	}

}



type scanProgressMsg core.ScanProgress

type scanFinishedMsg struct {

	items []core.JunkItem

	err   error

}



// waitForProgressOrFinish listens to both progress and scan completion
func waitForProgressOrFinish(progressChan <-chan core.ScanProgress, doneChan <-chan scanFinishedMsg) tea.Cmd {
	return func() tea.Msg {
		select {
		case progress, ok := <-progressChan:
			if !ok {
				// Progress channel closed, but we might still get the finish message
				return nil
			}
			return scanProgressMsg(progress)
		case finished := <-doneChan:
			return finished
		}
	}
}

func (m ScanModel) Init() tea.Cmd {
	// Create channels
	progressChan := make(chan core.ScanProgress, 10)
	doneChan := make(chan scanFinishedMsg, 1)

	// Store channels in model
	m.progressChan = progressChan
	m.doneChan = doneChan

	// Start the scanning goroutine
	go func() {
		// Get current working directory
		rootDir, err := os.Getwd()
		if err != nil {
			doneChan <- scanFinishedMsg{err: fmt.Errorf("could not get current working directory: %w", err)}
			close(progressChan)
			return
		}

		// Scan with progress reporting
		globalIgnorePaths := m.Config.IgnorePaths
		junkItems, err := core.ScanProjectJunk(rootDir, m.Config.Scan, globalIgnorePaths, m.Config, time.Now(), progressChan)

		// Close progress channel after scan completes
		close(progressChan)

		// Send result
		if err != nil {
			doneChan <- scanFinishedMsg{err: fmt.Errorf("scan failed: %w", err)}
		} else {
			doneChan <- scanFinishedMsg{items: junkItems}
		}
	}()

	return tea.Batch(
		m.spinner.Init(), // Start spinner animation
		waitForProgressOrFinish(m.progressChan, m.doneChan), // Wait for progress or completion
	)
}



type deleteResultMsg struct {



	path string



	err  error



}







func deleteSelectedItems(selectedItems []core.JunkItem) tea.Cmd {



	cmds := make([]tea.Cmd, len(selectedItems))



	for i, item := range selectedItems {



		item := item // Capture loop variable



		cmds[i] = func() tea.Msg {



			err := core.SafeDelete(item.Path)



			return deleteResultMsg{path: item.Path, err: err}



		}



	}



	return tea.Batch(cmds...)



}







func (m ScanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {







	var cmd tea.Cmd







	var cmds []tea.Cmd















	switch msg := msg.(type) {







	case tea.WindowSizeMsg:







		m.width = msg.Width







		m.height = msg.Height







	}















	if m.showConfirmation {







		// If confirmation is active, pass messages to it







		confirmModel, confirmCmd := m.confirmation.Update(msg)







		m.confirmation = confirmModel.(components.ConfirmationModel)







		cmds = append(cmds, confirmCmd)















		switch msg := msg.(type) {







		case components.ConfirmationMsg:







			m.showConfirmation = false







			if msg {







				// User confirmed deletion







				selectedJunkItems := []core.JunkItem{}







				for idx := range m.selected {







					selectedJunkItems = append(selectedJunkItems, m.items[idx])







				}







				cmds = append(cmds, deleteSelectedItems(selectedJunkItems))







			} else {







				// User canceled deletion







				// Optionally, show a message or just return







			}







			return m, tea.Batch(cmds...)







		}







	}















	if m.scanning {







		m.spinner, cmd = m.spinner.Update(msg)







		cmds = append(cmds, cmd)







	}















	switch msg := msg.(type) {







	case tea.KeyMsg:







		if m.scanning || m.showConfirmation {







			return m, tea.Batch(cmds...) // Ignore key presses while scanning or confirming







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







			if m.cursor < len(m.items) && m.items[m.cursor].RulesApplied {







				if _, ok := m.selected[m.cursor]; ok {







					delete(m.selected, m.cursor)







				} else {







					m.selected[m.cursor] = struct{}{}







				}







			}







		case "a": // Toggle all







			if len(m.selected) == len(m.items) {







				m.selected = make(map[int]struct{})







			} else {







				for i, item := range m.items {







					if item.RulesApplied {







						m.selected[i] = struct{}{}







					}







				}







			}

		case "s": // Toggle sort mode
			// Cycle through sort modes: None -> Size -> Date -> None
			switch m.currentSortMode {
			case sortByNone:
				m.currentSortMode = sortBySize
			case sortBySize:
				m.currentSortMode = sortByDate
			case sortByDate:
				m.currentSortMode = sortByNone
			}
			// Apply the new sort
			m.sortItems()
			// Reset cursor to top after sorting
			m.cursor = 0







		case "enter":







			if len(m.selected) > 0 {







				m.showConfirmation = true







				return m, m.confirmation.Init()







			}







			return m, nil







		case "esc", "q":







			return m, func() tea.Msg { return backToMenuMsg{} }







		}















	case scanProgressMsg:
		// Update progress tracking
		m.directoriesScanned = msg.DirectoriesScanned
		m.junkFoundMB = msg.JunkFoundMB
		m.junkItemsCount = msg.JunkItemsCount

		// Continue listening for more progress or the finish message
		return m, waitForProgressOrFinish(m.progressChan, m.doneChan)

	case scanFinishedMsg:







		m.scanning = false







		if msg.err != nil {







			return m, func() tea.Msg { return errMsg(msg.err) }







		}







		m.items = msg.items















	case deleteResultMsg:







		if msg.err != nil {







			return m, func() tea.Msg { return errMsg(fmt.Errorf("failed to delete %s: %w", msg.path, msg.err)) }







		}







		// Remove the deleted item from the list







		newItems := []core.JunkItem{}







		newSelected := make(map[int]struct{})







		for i, item := range m.items {







			if item.Path != msg.path {







				newItems = append(newItems, item)







				// Adjust selected indices if necessary







				if _, ok := m.selected[i]; ok {







					newSelected[len(newItems)-1] = struct{}{}







				}







			}







		}







		m.items = newItems







		m.selected = newSelected







		// Adjust cursor if it was pointing to the deleted item







		if m.cursor >= len(m.items) && len(m.items) > 0 {







			m.cursor = len(m.items) - 1







		} else if len(m.items) == 0 {







			m.cursor = 0







		}







	}















	return m, tea.Batch(cmds...)







}







var (







	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true).PaddingBottom(1)







	itemStyle   = lipgloss.NewStyle().PaddingLeft(2)







	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(0).Foreground(lipgloss.Color("10")).Bold(true)







	dimmedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("240"))







)















func (m ScanModel) View() string {







	if m.showConfirmation {







		return m.renderConfirmationView()







	}















	if m.scanning {







		// Display spinner with real-time statistics
		spinnerView := m.spinner.View()

		// Add real-time statistics
		stats := fmt.Sprintf("\nScanned %d directories, found %.1f MB junk (%d items matching rules)",
			m.directoriesScanned,
			m.junkFoundMB,
			m.junkItemsCount,
		)

		return spinnerView + stats







	}















	if len(m.items) == 0 {







		return "No junk items found. Press Esc/Q to go back."







	}















	s := headerStyle.Render("Found junk items:") + "\n"















	for i, item := range m.items {







		var style lipgloss.Style







		cursor := " "















		if m.cursor == i {







			cursor = ">"







			style = selectedItemStyle







		} else {







			style = itemStyle







		}















		checked := " "







		if _, ok := m.selected[i]; ok {







			checked = "x"







		}















		itemStr := fmt.Sprintf("[%s] %s (%.1f MB) (Last Modified %d days)", checked, item.Path, item.SizeMB, item.LastModifiedDays)















		if !item.RulesApplied {







			style = dimmedItemStyle







			itemStr += " [RULE SKIPPED]"







		}















		s += style.Render(fmt.Sprintf("%s %s", cursor, itemStr)) + "\n"







	}















	// Display current sort mode
	sortModeStr := "None"
	switch m.currentSortMode {
	case sortBySize:
		sortModeStr = "Size"
	case sortByDate:
		sortModeStr = "Date"
	}

	s += fmt.Sprintf("\n(Space to toggle, 'a' to toggle all, 's' to sort [%s], Enter to delete, Esc/Q to go back)", sortModeStr)







	return s







}







func (m ScanModel) renderConfirmationView() string {



	return lipgloss.Place(



		m.width,



		m.height,



		lipgloss.Center,



		lipgloss.Center,



		m.confirmation.View(),



	)



}

// sortItems sorts the items array based on the current sort mode
func (m *ScanModel) sortItems() {
	switch m.currentSortMode {
	case sortBySize:
		sort.SliceStable(m.items, func(i, j int) bool {
			return m.items[i].SizeMB > m.items[j].SizeMB // Descending by size
		})
	case sortByDate:
		sort.SliceStable(m.items, func(i, j int) bool {
			return m.items[i].LastModifiedDays > m.items[j].LastModifiedDays // Descending by days (older first)
		})
	case sortByNone:
		// No sorting, keep original order
	}
}
