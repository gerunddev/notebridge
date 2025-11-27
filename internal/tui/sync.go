package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	spinnerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	highlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
)

// SyncResult holds the result of a sync operation
type SyncResult struct {
	FilesProcessed int
	Errors         []error
	Duration       time.Duration
	Success        bool
}

// syncModel is the Bubble Tea model for the sync progress display
type syncModel struct {
	spinner      spinner.Model
	status       string
	filesScanned int
	filesSynced  int
	complete     bool
	result       *SyncResult
	err          error
}

// SyncMsg is sent when sync completes
type SyncMsg struct {
	Result *SyncResult
	Err    error
}

// InitSyncModel creates a new sync progress model
func InitSyncModel() syncModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return syncModel{
		spinner: s,
		status:  "Initializing sync...",
	}
}

func (m syncModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m syncModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case SyncMsg:
		m.complete = true
		m.result = msg.Result
		m.err = msg.Err
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m syncModel) View() string {
	if m.complete {
		if m.err != nil {
			return errorStyle.Render("✗ Sync failed: " + m.err.Error()) + "\n"
		}

		if m.result.FilesProcessed == 0 {
			return successStyle.Render("✓ Nothing to sync") + "\n" +
				helpStyle.Render(fmt.Sprintf("Completed in %v", m.result.Duration.Round(time.Millisecond))) + "\n"
		}

		msg := successStyle.Render(fmt.Sprintf("✓ Synced %d file(s)", m.result.FilesProcessed))
		if len(m.result.Errors) > 0 {
			msg += ", " + errorStyle.Render(fmt.Sprintf("%d error(s)", len(m.result.Errors)))
		}
		msg += "\n" + helpStyle.Render(fmt.Sprintf("Completed in %v", m.result.Duration.Round(time.Millisecond))) + "\n"

		return msg
	}

	return fmt.Sprintf("\n%s %s\n\n", m.spinner.View(), m.status)
}

// UpdateStatus updates the status message
type UpdateStatusMsg string

// UpdateFilesScanned updates the scanned file count
type UpdateFilesScannedMsg int

// UpdateFilesSynced updates the synced file count
type UpdateFilesSyncedMsg int

func (m syncModel) UpdateStatus(status string) syncModel {
	m.status = status
	return m
}
