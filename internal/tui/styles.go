package tui

import "github.com/charmbracelet/lipgloss"

var (
	// SuccessStyle renders success messages in green.
	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	// ErrorStyle renders error messages in red.
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	// WarningStyle renders warning messages in yellow.
	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Bold(true)

	// InfoStyle renders informational messages in cyan.
	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14"))

	// TableHeaderStyle renders table column headers in bold white.
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15")).
				PaddingRight(2)

	// TableCellStyle renders table cell values with padding.
	TableCellStyle = lipgloss.NewStyle().
			PaddingRight(2)

	// TableBorderStyle renders table borders in dark grey.
	TableBorderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8"))
)
