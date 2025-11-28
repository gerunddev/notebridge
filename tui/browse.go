package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerunddev/notebridge/diff"
	"github.com/gerunddev/notebridge/state"
	"github.com/gerunddev/notebridge/styles"
)

// BrowseData holds all tracked files and their status
type BrowseData struct {
	Files []FileInfo
}

// FileInfo represents a tracked file pair with its status
type FileInfo struct {
	BaseName   string
	OrgPath    string
	MdPath     string
	Status     string // "synced", "pending-org", "pending-md", "conflict"
	StatusIcon string // "✓", "→", "←", "⚠"
	HasOrgFile bool
	HasMdFile  bool
}

// BrowseMsg is sent when browse data is ready
type BrowseMsg struct {
	Data *BrowseData
	Err  error
}

// DiffMsg is sent when diff preview is ready
type DiffMsg struct {
	Content string
	Format  diff.Format
	Err     error
}

// RefreshBrowseMsg triggers a browse data refresh
type RefreshBrowseMsg struct{}

type browseModel struct {
	table         table.Model
	viewport      viewport.Model
	data          *BrowseData
	err           error
	ready         bool
	showingDiff   bool
	showingPrompt bool
	diffContent   string
	diffFormat    diff.Format
	width         int
	height        int
	selectedFile  *FileInfo
	// Dependencies for resolution
	orgDir      string
	obsidianDir string
	state       *state.State
	resolveFunc func(orgPath, mdPath, direction string) error
	refreshFunc func()
}

// InitBrowseModel creates a new file browser model
func InitBrowseModel(orgDir, obsidianDir string, st *state.State, resolveFunc func(string, string, string) error, refreshFunc func()) browseModel {
	columns := []table.Column{
		{Title: "File", Width: 50},
		{Title: "Status", Width: 20},
		{Title: "Org", Width: 8},
		{Title: "Markdown", Width: 8},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	ts := table.DefaultStyles()
	ts.Header = ts.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(styles.Border)).
		BorderBottom(true).
		Bold(false)
	ts.Selected = ts.Selected.
		Foreground(lipgloss.Color(styles.Background)).
		Background(lipgloss.Color(styles.Yellow)).
		Bold(false)
	t.SetStyles(ts)

	vp := viewport.New(100, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(styles.Border)).
		Padding(1)

	return browseModel{
		table:       t,
		viewport:    vp,
		orgDir:      orgDir,
		obsidianDir: obsidianDir,
		state:       st,
		resolveFunc: resolveFunc,
		refreshFunc: refreshFunc,
	}
}

func (m browseModel) Init() tea.Cmd {
	return nil
}

func (m browseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(msg.Height - 10)
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 6

	case tea.KeyMsg:
		if m.showingPrompt {
			// In resolution prompt
			switch msg.String() {
			case "1", "2", "3", "4":
				if m.selectedFile != nil {
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
					m.showingDiff = false
					return m, m.performResolution(action)
				}
				return m, nil
			case "esc":
				m.showingPrompt = false
				return m, nil
			}
		} else if m.showingDiff {
			// In diff view
			switch msg.String() {
			case "q", "esc":
				m.showingDiff = false
				return m, nil
			case "r":
				// Show resolution prompt
				if m.selectedFile != nil && (m.selectedFile.Status == "conflict" || m.selectedFile.Status == "org → md" || m.selectedFile.Status == "md → org") {
					m.showingPrompt = true
				}
				return m, nil
			case "up", "k":
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			case "down", "j":
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}
		} else {
			// In table view
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				m.table, cmd = m.table.Update(msg)
				return m, cmd
			case "down", "j":
				m.table, cmd = m.table.Update(msg)
				return m, cmd
			case "enter", "d":
				// Show diff for selected file
				if m.data != nil && len(m.data.Files) > 0 {
					selectedIdx := m.table.Cursor()
					if selectedIdx < len(m.data.Files) {
						m.selectedFile = &m.data.Files[selectedIdx]
						m.showingDiff = true
						return m, m.loadDiff()
					}
				}
				return m, nil
			}
		}

	case BrowseMsg:
		m.ready = true
		m.data = msg.Data
		m.err = msg.Err

		if m.data != nil {
			// Build table rows
			rows := []table.Row{}
			for _, file := range m.data.Files {
				orgStatus := "✗"
				if file.HasOrgFile {
					orgStatus = "✓"
				}
				mdStatus := "✗"
				if file.HasMdFile {
					mdStatus = "✓"
				}

				statusText := fmt.Sprintf("%s %s", file.StatusIcon, file.Status)
				rows = append(rows, table.Row{
					file.BaseName,
					statusText,
					orgStatus,
					mdStatus,
				})
			}
			m.table.SetRows(rows)
		}

		return m, nil

	case DiffMsg:
		m.diffContent = msg.Content
		m.diffFormat = msg.Format
		m.viewport.SetContent(m.diffContent)
		m.viewport.GotoTop()
		return m, nil

	case RefreshBrowseMsg:
		// Trigger browse data refresh
		if m.refreshFunc != nil {
			go m.refreshFunc()
		}
		return m, nil
	}

	return m, nil
}

func (m browseModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("NoteBridge File Browser"))
	b.WriteString("\n\n")

	if m.err != nil {
		return errorStyle.Render("✗ Error: "+m.err.Error()) + "\n"
	}

	if !m.ready || m.data == nil {
		return b.String()
	}

	if m.showingPrompt {
		// Show resolution prompt
		b.WriteString(highlightStyle.Render("Choose resolution action:"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  [1] Use %s version\n", highlightStyle.Render("Org")))
		b.WriteString(fmt.Sprintf("  [2] Use %s version\n", highlightStyle.Render("Markdown")))
		b.WriteString(fmt.Sprintf("  [3] %s (sync newer file)\n", highlightStyle.Render("Last-write-wins")))
		b.WriteString(fmt.Sprintf("  [4] %s\n", helpStyle.Render("Skip")))
		b.WriteString(fmt.Sprintf("\n  File: %s\n", valueStyle.Render(m.selectedFile.BaseName)))
		if m.selectedFile.Status == "conflict" {
			b.WriteString(fmt.Sprintf("  %s\n", errorStyle.Render("⚠ Both versions have changed")))
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("1-4 choose • esc cancel"))
		b.WriteString("\n")
	} else if m.showingDiff {
		// Show diff view
		b.WriteString(labelStyle.Render(fmt.Sprintf("Diff Preview: %s", m.selectedFile.BaseName)))
		b.WriteString("\n\n")
		b.WriteString(m.viewport.View())
		b.WriteString("\n\n")
		// Show resolve option if file needs resolution
		if m.selectedFile != nil && (m.selectedFile.Status == "conflict" || m.selectedFile.Status == "org → md" || m.selectedFile.Status == "md → org") {
			b.WriteString(helpStyle.Render("↑/k up • ↓/j down • r resolve • esc/q back"))
		} else {
			b.WriteString(helpStyle.Render("↑/k up • ↓/j down • esc/q back"))
		}
		b.WriteString("\n")
	} else {
		// Show table view
		b.WriteString(labelStyle.Render(fmt.Sprintf("Tracked Files: %d", len(m.data.Files))))
		b.WriteString("\n\n")
		b.WriteString(tableStyle.Render(m.table.View()))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("↑/k up • ↓/j down • enter/d diff • q quit"))
		b.WriteString("\n")
	}

	return b.String()
}

// loadDiff creates a command that loads the diff for the selected file
// Automatically determines format based on sync direction (destination format)
func (m browseModel) loadDiff() tea.Cmd {
	return func() tea.Msg {
		if m.selectedFile == nil {
			return DiffMsg{
				Content: "No file selected",
				Err:     fmt.Errorf("no file selected"),
			}
		}

		// Build full paths
		orgPath := filepath.Join(m.orgDir, m.selectedFile.OrgPath)
		mdPath := filepath.Join(m.obsidianDir, m.selectedFile.MdPath)

		// Determine format based on sync direction (destination format)
		format, err := diff.DefaultFormat(orgPath, mdPath)
		if err != nil {
			// Fallback to markdown on error
			format = diff.FormatMarkdown
		}

		// Generate diff with destination format
		diffContent, err := diff.Generate(orgPath, mdPath, m.state, format)
		if err != nil {
			return DiffMsg{
				Content: fmt.Sprintf("Error generating diff: %s", err.Error()),
				Format:  format,
				Err:     err,
			}
		}

		return DiffMsg{
			Content: diffContent,
			Format:  format,
			Err:     nil,
		}
	}
}

// performResolution creates a command that performs the file sync
func (m browseModel) performResolution(action ResolutionAction) tea.Cmd {
	return func() tea.Msg {
		if m.resolveFunc == nil || m.selectedFile == nil {
			return RefreshBrowseMsg{}
		}

		var direction string
		switch action {
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
		orgPath := filepath.Join(m.orgDir, m.selectedFile.OrgPath)
		mdPath := filepath.Join(m.obsidianDir, m.selectedFile.MdPath)

		// Perform the sync
		if err := m.resolveFunc(orgPath, mdPath, direction); err != nil {
			// TODO: Show error to user
			_ = err
		}

		// Return refresh message to reload browse data
		return RefreshBrowseMsg{}
	}
}
