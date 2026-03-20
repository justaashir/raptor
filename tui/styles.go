package tui

import (
	"raptor/model"

	"github.com/charmbracelet/lipgloss"
)

// Vibrant palette (Dracula-inspired but more lively)
var (
	colorBg        = lipgloss.Color("#282a36")
	colorLine      = lipgloss.Color("#44475a")
	colorFg        = lipgloss.Color("#f8f8f2")
	colorComment   = lipgloss.Color("#6272a4")
	colorCyan      = lipgloss.Color("#8be9fd")
	colorGreen     = lipgloss.Color("#50fa7b")
	colorOrange    = lipgloss.Color("#ffb86c")
	colorPink      = lipgloss.Color("#ff79c6")
	colorPurple    = lipgloss.Color("#bd93f9")
	colorRed       = lipgloss.Color("#ff5555")
	colorYellow    = lipgloss.Color("#f1fa8c")
	colorBrightPurple = lipgloss.Color("#d6acff")
	colorGold      = lipgloss.Color("#ffd700")
)

// StatusColor returns a color for each status.
func StatusColor(s model.Status) lipgloss.Color {
	switch s {
	case model.Todo:
		return colorOrange
	case model.InProgress:
		return colorCyan
	case model.Done:
		return colorGreen
	case model.Closed:
		return colorRed
	default:
		return colorComment
	}
}

// StatusIcon returns an emoji for each status.
func StatusIcon(s model.Status) string {
	switch s {
	case model.Todo:
		return "📋"
	case model.InProgress:
		return "⚡"
	case model.Done:
		return "✅"
	case model.Closed:
		return "🔒"
	default:
		return "📄"
	}
}

// StatusStar returns a star emoji for open tickets.
func StatusStar(s model.Status) string {
	switch s {
	case model.Todo, model.InProgress:
		return "⭐"
	default:
		return "  "
	}
}

// Pane styles
var (
	FocusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPurple)

	UnfocusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorComment)

	// Column header bar (purple background like beads_viewer)
	ColumnHeaderStyle = lipgloss.NewStyle().
				Background(colorPurple).
				Foreground(colorBg).
				Bold(true).
				Padding(0, 1)

	StatusBarStyle = lipgloss.NewStyle().
			Background(colorLine).
			Foreground(colorFg).
			Padding(0, 1)

	DetailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPink)

	DetailMetaKeyStyle = lipgloss.NewStyle().
				Foreground(colorPurple).
				Bold(true)

	DetailMetaValueStyle = lipgloss.NewStyle().
				Foreground(colorFg)
)
