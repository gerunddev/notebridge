package styles

import "github.com/charmbracelet/lipgloss"

// Monokai Pro color palette
const (
	// Base colors
	Background = "#2D2A2E"
	Foreground = "#FCFCFA"

	// Accent colors
	Red     = "#FF6188" // Errors, danger
	Orange  = "#FC9867" // Warnings
	Yellow  = "#FFD866" // Highlights
	Green   = "#A9DC76" // Success
	Cyan    = "#78DCE8" // Info
	Blue    = "#AB9DF2" // Links
	Magenta = "#FF6188" // Titles, emphasis

	// UI colors
	Comment = "#727072" // Dim text, help
	Border  = "#5B595C" // Borders, separators
)

// Common styles
var (
	SuccessStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(Green))
	ErrorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(Red))
	WarningStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(Orange))
	DimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color(Comment))
	TitleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Magenta))
	HighlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(Yellow)).Bold(true)
	SpinnerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(Magenta))
	HelpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color(Comment))

	// Table/list styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(Magenta))

	TableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(Border))

	SelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Background)).
			Background(lipgloss.Color(Yellow))

	NormalTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Foreground))
)
