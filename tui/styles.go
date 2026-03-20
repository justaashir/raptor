package tui

import (
	"raptor/model"

	"github.com/charmbracelet/lipgloss"
)

// Dracula palette
var (
	draculaBg      = lipgloss.Color("#282a36")
	draculaLine    = lipgloss.Color("#44475a")
	draculaFg      = lipgloss.Color("#f8f8f2")
	draculaComment = lipgloss.Color("#6272a4")
	draculaCyan    = lipgloss.Color("#8be9fd")
	draculaGreen   = lipgloss.Color("#50fa7b")
	draculaOrange  = lipgloss.Color("#ffb86c")
	draculaPink    = lipgloss.Color("#ff79c6")
	draculaPurple  = lipgloss.Color("#bd93f9")
	draculaRed     = lipgloss.Color("#ff5555")
	draculaYellow  = lipgloss.Color("#f1fa8c")
)

// Status colors — Dracula
func StatusColor(s model.Status) lipgloss.Color {
	switch s {
	case model.Todo:
		return draculaOrange
	case model.InProgress:
		return draculaCyan
	case model.Done:
		return draculaGreen
	case model.Closed:
		return draculaRed
	default:
		return draculaComment
	}
}

// StatusIcon returns a colored icon for each status.
func StatusIcon(s model.Status) string {
	switch s {
	case model.Todo:
		return "○"
	case model.InProgress:
		return "◉"
	case model.Done:
		return "✓"
	case model.Closed:
		return "✗"
	default:
		return "·"
	}
}

// Pane styles — Dracula
var (
	FocusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(draculaPurple)

	UnfocusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(draculaComment)

	StatusBarStyle = lipgloss.NewStyle().
			Background(draculaLine).
			Foreground(draculaFg).
			Padding(0, 1)

	DetailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(draculaPink)

	DetailMetaKeyStyle = lipgloss.NewStyle().
				Foreground(draculaComment).
				Bold(true)

	DetailMetaValueStyle = lipgloss.NewStyle().
				Foreground(draculaFg)
)
