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
	colorYellow = lipgloss.Color("#f1fa8c")
)

// statusPalette cycles through colors for dynamic statuses beyond the known ones.
var statusPalette = []lipgloss.Color{
	colorOrange, colorCyan, colorGreen, colorPink, colorPurple, colorYellow,
}

// StatusColor returns a color for each status.
func StatusColor(s model.Status) lipgloss.Color {
	switch s {
	case model.Todo:
		return colorOrange
	case model.InProgress:
		return colorCyan
	case model.Done:
		return colorComment
	default:
		// Cycle through palette based on status name hash
		h := 0
		for _, c := range string(s) {
			h += int(c)
		}
		return statusPalette[h%len(statusPalette)]
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
	default:
		return "📄"
	}
}

// StatusStar returns a star emoji for open tickets.
func StatusStar(s model.Status) string {
	switch s {
	case model.Done:
		return "  "
	default:
		return "⭐"
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

	DetailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPink)

	DetailMetaKeyStyle = lipgloss.NewStyle().
				Foreground(colorPurple).
				Bold(true)

)
