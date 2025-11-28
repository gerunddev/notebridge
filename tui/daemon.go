package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// DaemonData holds daemon status information
type DaemonData struct {
	Running      bool
	PID          int
	StartTime    time.Time
	LastSyncTime time.Time
	FilesSynced  int
	LogLines     []string
}

// DaemonMsg is sent when daemon data is ready
type DaemonMsg struct {
	Data *DaemonData
	Err  error
}

// TickMsg triggers a periodic refresh
type TickMsg time.Time

type daemonModel struct {
	data  *DaemonData
	err   error
	ready bool
}

// InitDaemonModel creates a new daemon status model
func InitDaemonModel() daemonModel {
	return daemonModel{}
}

func (m daemonModel) Init() tea.Cmd {
	return tick()
}

func (m daemonModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case TickMsg:
		// Request data refresh
		return m, tea.Batch(tick(), tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return TickMsg(t)
		}))

	case DaemonMsg:
		m.ready = true
		m.data = msg.Data
		m.err = msg.Err
		return m, nil
	}

	return m, nil
}

func (m daemonModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("NoteBridge Daemon Dashboard"))
	b.WriteString("\n\n")

	if m.err != nil {
		return errorStyle.Render("✗ Error: "+m.err.Error()) + "\n"
	}

	if !m.ready || m.data == nil {
		return b.String()
	}

	// Daemon Status
	b.WriteString(labelStyle.Render("Daemon Status"))
	b.WriteString("\n")
	if m.data.Running {
		uptime := time.Since(m.data.StartTime).Round(time.Second)
		b.WriteString(fmt.Sprintf("  Status: %s\n", successStyle.Render("● Running")))
		b.WriteString(fmt.Sprintf("  PID:    %s\n", valueStyle.Render(fmt.Sprintf("%d", m.data.PID))))
		b.WriteString(fmt.Sprintf("  Uptime: %s\n", valueStyle.Render(uptime.String())))
	} else {
		b.WriteString(fmt.Sprintf("  Status: %s\n", helpStyle.Render("○ Not running")))
	}
	b.WriteString("\n")

	// Sync Info
	b.WriteString(labelStyle.Render("Sync Information"))
	b.WriteString("\n")
	if m.data.Running {
		if !m.data.LastSyncTime.IsZero() {
			timeSince := time.Since(m.data.LastSyncTime).Round(time.Second)
			b.WriteString(fmt.Sprintf("  Last sync:    %s ago\n", valueStyle.Render(timeSince.String())))
			b.WriteString(fmt.Sprintf("  Files synced: %s\n", valueStyle.Render(fmt.Sprintf("%d", m.data.FilesSynced))))
		} else {
			b.WriteString(fmt.Sprintf("  %s\n", helpStyle.Render("No sync completed yet")))
		}
	} else {
		b.WriteString(fmt.Sprintf("  %s\n", helpStyle.Render("Daemon not running")))
	}
	b.WriteString("\n")

	// Log Tail
	b.WriteString(labelStyle.Render("Recent Logs"))
	b.WriteString("\n")
	if len(m.data.LogLines) > 0 {
		for _, line := range m.data.LogLines {
			b.WriteString("  " + line + "\n")
		}
	} else {
		b.WriteString(helpStyle.Render("  No logs available"))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Help
	b.WriteString(helpStyle.Render("q quit • auto-refresh: 2s"))
	b.WriteString("\n")

	return b.String()
}

// tick returns a command that sends a TickMsg
func tick() tea.Cmd {
	return func() tea.Msg {
		return TickMsg(time.Now())
	}
}
