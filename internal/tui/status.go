package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))

	tableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
)

// StatusData holds all the information for the status display
type StatusData struct {
	OrgDir         string
	ObsidianDir    string
	Interval       time.Duration
	OrgFileCount   int
	MdFileCount    int
	TrackedPairs   int
	PendingOrg     []string
	PendingMd      []string
	Conflicts      []string
	IDMapCount     int
	Scanning       bool
}

// StatusMsg is sent when status data is ready
type StatusMsg struct {
	Data *StatusData
	Err  error
}

// ResolutionAction represents a conflict resolution choice
type ResolutionAction string

const (
	UseOrg         ResolutionAction = "org"
	UseMarkdown    ResolutionAction = "markdown"
	LastWriteWins  ResolutionAction = "last-write-wins"
	Skip           ResolutionAction = "skip"
)

// ResolveMsg is sent when a file resolution is chosen
type ResolveMsg struct {
	Action   ResolutionAction
	FileRow  fileRow
}

// RefreshStatusMsg triggers a status refresh
type RefreshStatusMsg struct{}

type statusModel struct {
	spinner        spinner.Model
	data           *StatusData
	table          table.Model
	err            error
	scanning       bool
	ready          bool
	width          int
	height         int
	fileRows       []fileRow // Track file info for each row
	showingPrompt  bool
	selectedAction string
	// Dependencies for resolution
	orgDir      string
	obsidianDir string
	resolveFunc func(orgPath, mdPath, direction string) error
	refreshFunc func()
}

// fileRow tracks the file information for each table row
type fileRow struct {
	baseName    string // filename without extension
	orgPath     string
	mdPath      string
	isConflict  bool
	fileType    string // "org", "md", or "conflict"
}

// InitStatusModel creates a new status display model
func InitStatusModel(orgDir, obsidianDir string, resolveFunc func(string, string, string) error, refreshFunc func()) statusModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	columns := []table.Column{
		{Title: "File", Width: 40},
		{Title: "Type", Width: 10},
		{Title: "Status", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	ts := table.DefaultStyles()
	ts.Header = ts.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	ts.Selected = ts.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(ts)

	return statusModel{
		spinner:     s,
		scanning:    true,
		table:       t,
		orgDir:      orgDir,
		obsidianDir: obsidianDir,
		resolveFunc: resolveFunc,
		refreshFunc: refreshFunc,
	}
}

func (m statusModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m statusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		case "down", "j":
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		case "enter":
			// Show resolution prompt for selected file
			if len(m.fileRows) > 0 {
				selectedIdx := m.table.Cursor()
				if selectedIdx < len(m.fileRows) {
					m.showingPrompt = true
				}
			}
			return m, nil
		case "1", "2", "3", "4":
			// Handle resolution choice when prompt is showing
			if m.showingPrompt && len(m.fileRows) > 0 {
				selectedIdx := m.table.Cursor()
				if selectedIdx < len(m.fileRows) {
					fileRow := m.fileRows[selectedIdx]
					var action ResolutionAction
					switch msg.String() {
					case "1":
						action = UseOrg
					case "2":
						action = UseMarkdown
					case "3":
						action = LastWriteWins
					case "4":
						action = Skip
					}
					m.showingPrompt = false
					return m, func() tea.Msg {
						return ResolveMsg{
							Action:  action,
							FileRow: fileRow,
						}
					}
				}
			}
			return m, nil
		case "esc":
			// Cancel resolution prompt
			if m.showingPrompt {
				m.showingPrompt = false
			}
			return m, nil
		}

	case ResolveMsg:
		// Perform the sync with the chosen resolution
		return m, m.performResolution(msg)

	case RefreshStatusMsg:
		// Trigger status refresh
		if m.refreshFunc != nil {
			go m.refreshFunc()
		}
		return m, nil

	case StatusMsg:
		m.scanning = false
		m.ready = true
		m.data = msg.Data
		m.err = msg.Err

		if m.data != nil {
			// Build table rows - conflicts first, then non-conflicts
			rows := []table.Row{}
			m.fileRows = []fileRow{}

			// Track which files we've already added as conflicts
			conflictSet := make(map[string]bool)
			for _, c := range m.data.Conflicts {
				conflictSet[c] = true
			}

			// Add conflicts as single rows
			for _, c := range m.data.Conflicts {
				rows = append(rows, table.Row{c, "Both", "⚠ Conflict"})
				m.fileRows = append(m.fileRows, fileRow{
					baseName:   c,
					orgPath:    c + ".org",
					mdPath:     c + ".md",
					isConflict: true,
					fileType:   "conflict",
				})
			}

			// Add non-conflicting org files
			for _, f := range m.data.PendingOrg {
				baseName := strings.TrimSuffix(f, ".org")
				if !conflictSet[baseName] {
					rows = append(rows, table.Row{f, "Org", "Changed"})
					m.fileRows = append(m.fileRows, fileRow{
						baseName:   baseName,
						orgPath:    f,
						mdPath:     "",
						isConflict: false,
						fileType:   "org",
					})
				}
			}

			// Add non-conflicting md files
			for _, f := range m.data.PendingMd {
				baseName := strings.TrimSuffix(f, ".md")
				if !conflictSet[baseName] {
					rows = append(rows, table.Row{f, "Markdown", "Changed"})
					m.fileRows = append(m.fileRows, fileRow{
						baseName:   baseName,
						orgPath:    "",
						mdPath:     f,
						isConflict: false,
						fileType:   "md",
					})
				}
			}

			m.table.SetRows(rows)
		}

		return m, nil

	case spinner.TickMsg:
		if m.scanning {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m statusModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("NoteBridge Status"))
	b.WriteString("\n\n")

	if m.err != nil {
		return errorStyle.Render("✗ Error: " + m.err.Error()) + "\n"
	}

	if m.scanning {
		b.WriteString(fmt.Sprintf("%s Scanning directories...\n", m.spinner.View()))
		return b.String()
	}

	if !m.ready || m.data == nil {
		return b.String()
	}

	// Configuration
	b.WriteString(labelStyle.Render("Configuration"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Org directory:      %s\n", valueStyle.Render(m.data.OrgDir)))
	b.WriteString(fmt.Sprintf("  Obsidian directory: %s\n", valueStyle.Render(m.data.ObsidianDir)))
	b.WriteString(fmt.Sprintf("  Sync interval:      %s\n", valueStyle.Render(m.data.Interval.String())))
	b.WriteString("\n")

	// Files
	b.WriteString(labelStyle.Render("Files"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Org files:      %s\n", valueStyle.Render(fmt.Sprintf("%d", m.data.OrgFileCount))))
	b.WriteString(fmt.Sprintf("  Markdown files: %s\n", valueStyle.Render(fmt.Sprintf("%d", m.data.MdFileCount))))
	b.WriteString(fmt.Sprintf("  Tracked pairs:  %s\n", valueStyle.Render(fmt.Sprintf("%d", m.data.TrackedPairs))))
	b.WriteString("\n")

	// Summary
	b.WriteString(labelStyle.Render("Pending Changes"))
	b.WriteString("\n")

	totalPending := len(m.data.PendingOrg) + len(m.data.PendingMd)
	if totalPending == 0 {
		b.WriteString(fmt.Sprintf("  %s\n", successStyle.Render("✓ No pending changes")))
	} else {
		if len(m.data.PendingOrg) > 0 {
			b.WriteString(fmt.Sprintf("  %s\n", highlightStyle.Render(fmt.Sprintf("● %d org file(s) changed", len(m.data.PendingOrg)))))
		}
		if len(m.data.PendingMd) > 0 {
			b.WriteString(fmt.Sprintf("  %s\n", highlightStyle.Render(fmt.Sprintf("● %d markdown file(s) changed", len(m.data.PendingMd)))))
		}
	}
	b.WriteString("\n")

	// Conflicts summary
	b.WriteString(labelStyle.Render("Conflicts"))
	b.WriteString("\n")
	if len(m.data.Conflicts) == 0 {
		b.WriteString(fmt.Sprintf("  %s\n", successStyle.Render("✓ No conflicts")))
	} else {
		b.WriteString(fmt.Sprintf("  %s\n", errorStyle.Render(fmt.Sprintf("✗ %d potential conflict(s)", len(m.data.Conflicts)))))
	}
	b.WriteString("\n")

	// Interactive table
	if totalPending > 0 {
		b.WriteString(labelStyle.Render("File Details"))
		b.WriteString("\n")
		b.WriteString(tableStyle.Render(m.table.View()))
		b.WriteString("\n\n")
	}

	// ID mappings
	b.WriteString(labelStyle.Render("ID Mappings"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s\n", valueStyle.Render(fmt.Sprintf("%d org-roam IDs tracked", m.data.IDMapCount))))
	b.WriteString("\n")

	// Resolution prompt (if showing)
	if m.showingPrompt && len(m.fileRows) > 0 {
		selectedIdx := m.table.Cursor()
		if selectedIdx < len(m.fileRows) {
			fileRow := m.fileRows[selectedIdx]
			b.WriteString(highlightStyle.Render("Choose resolution action:"))
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("  [1] Use %s version\n", highlightStyle.Render("Org")))
			b.WriteString(fmt.Sprintf("  [2] Use %s version\n", highlightStyle.Render("Markdown")))
			b.WriteString(fmt.Sprintf("  [3] %s (sync newer file)\n", highlightStyle.Render("Last-write-wins")))
			b.WriteString(fmt.Sprintf("  [4] %s\n", helpStyle.Render("Skip")))
			b.WriteString(fmt.Sprintf("\n  File: %s\n", valueStyle.Render(fileRow.baseName)))
			if fileRow.isConflict {
				b.WriteString(fmt.Sprintf("  %s\n", errorStyle.Render("⚠ Both versions have changed")))
			}
			b.WriteString("\n")
			b.WriteString(helpStyle.Render("1-4 choose • esc cancel"))
			b.WriteString("\n")
			return b.String()
		}
	}

	// Help text (always show)
	if totalPending > 0 {
		b.WriteString(helpStyle.Render("↑/k up • ↓/j down • enter resolve • q/ctrl+c quit"))
	} else {
		b.WriteString(helpStyle.Render("q/ctrl+c quit"))
	}
	b.WriteString("\n")

	return b.String()
}

// performResolution creates a command that performs the file sync
func (m statusModel) performResolution(msg ResolveMsg) tea.Cmd {
	return func() tea.Msg {
		if m.resolveFunc == nil {
			return RefreshStatusMsg{}
		}

		fileRow := msg.FileRow
		var direction string
		switch msg.Action {
		case UseOrg:
			direction = "org"
		case UseMarkdown:
			direction = "markdown"
		case LastWriteWins:
			direction = "last-write-wins"
		case Skip:
			direction = "skip"
		}

		// Build full paths
		orgPath := filepath.Join(m.orgDir, fileRow.orgPath)
		mdPath := filepath.Join(m.obsidianDir, fileRow.mdPath)

		// Perform the sync
		if err := m.resolveFunc(orgPath, mdPath, direction); err != nil {
			// TODO: Show error to user
			_ = err
		}

		// Return refresh message to reload status
		return RefreshStatusMsg{}
	}
}
