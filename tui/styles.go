package tui

import (
	"raptor/model"
	"strings"

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

// OverlayOnBackground renders content inside a centered floating box
// composited on top of the background string so the background remains visible.
func OverlayOnBackground(content string, boxW, boxH int, bg string, termW, termH int) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurple).
		Background(colorBg).
		Foreground(colorFg).
		Width(boxW).
		Height(boxH).
		Padding(1, 2).
		Render(content)

	boxLines := strings.Split(box, "\n")
	bgLines := strings.Split(bg, "\n")

	// Pad background to fill terminal height
	for len(bgLines) < termH {
		bgLines = append(bgLines, "")
	}

	// Compute centered position
	boxRenderedW := lipgloss.Width(box)
	boxRenderedH := len(boxLines)
	startY := (termH - boxRenderedH) / 2
	startX := (termW - boxRenderedW) / 2
	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	// Overlay box lines onto background
	for i, bLine := range boxLines {
		bgIdx := startY + i
		if bgIdx >= len(bgLines) {
			break
		}
		bgLine := bgLines[bgIdx]
		// Pad background line to startX with spaces
		bgW := lipgloss.Width(bgLine)
		var left string
		if bgW >= startX {
			left = truncateToWidth(bgLine, startX)
		} else {
			left = bgLine + strings.Repeat(" ", startX-bgW)
		}
		// Right side of background after the box
		rightStart := startX + lipgloss.Width(bLine)
		var right string
		if bgW > rightStart {
			// We need the tail of the background line
			// Use a simple approach: pad and slice
			right = truncateRight(bgLine, rightStart)
		}
		bgLines[bgIdx] = left + bLine + right
	}

	return strings.Join(bgLines[:termH], "\n")
}

// truncateRight returns the portion of s after skipW display columns.
func truncateRight(s string, skipW int) string {
	runes := []rune(s)
	w := 0
	for i, r := range runes {
		w += lipgloss.Width(string(r))
		if w > skipW {
			return string(runes[i:])
		}
	}
	return ""
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
