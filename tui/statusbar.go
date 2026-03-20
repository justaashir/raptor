package tui

import (
	"fmt"
	"raptor/model"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// focusPane identifies which pane is focused.
type focusPane int

const (
	focusList focusPane = iota
	focusDetail
)

// CountByStatus counts tickets grouped by status.
func CountByStatus(tickets []model.Ticket) map[model.Status]int {
	counts := make(map[model.Status]int)
	for _, t := range tickets {
		counts[t.Status]++
	}
	return counts
}

// RenderStatusBar renders the bottom status bar with Dracula colors.
func RenderStatusBar(tickets []model.Ticket, boardName string, focus focusPane, width int) string {
	counts := CountByStatus(tickets)

	// Filter badge
	filterBadge := lipgloss.NewStyle().
		Background(draculaPurple).
		Foreground(lipgloss.Color("#282a36")).
		Bold(true).
		Padding(0, 1).
		Render("ALL")

	// Ticket counts with emojis
	left := fmt.Sprintf("%s  %s  %d tickets  %s %s %s %s",
		filterBadge,
		lipgloss.NewStyle().Foreground(draculaPink).Bold(true).Render(boardName),
		len(tickets),
		lipgloss.NewStyle().Foreground(StatusColor(model.Todo)).Render(
			fmt.Sprintf("📋%d", counts[model.Todo])),
		lipgloss.NewStyle().Foreground(StatusColor(model.InProgress)).Render(
			fmt.Sprintf("🔧%d", counts[model.InProgress])),
		lipgloss.NewStyle().Foreground(StatusColor(model.Done)).Render(
			fmt.Sprintf("✅%d", counts[model.Done])),
		lipgloss.NewStyle().Foreground(StatusColor(model.Closed)).Render(
			fmt.Sprintf("🔒%d", counts[model.Closed])),
	)

	// Right side: keybind hints with separators
	keyStyle := lipgloss.NewStyle().Foreground(draculaCyan).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(draculaComment)
	sep := lipgloss.NewStyle().Foreground(draculaComment).Render(" │ ")

	hints := []string{
		fmt.Sprintf("%s %s", keyStyle.Render("j/k"), descStyle.Render("nav")),
		fmt.Sprintf("%s %s", keyStyle.Render("/"), descStyle.Render("filter")),
		fmt.Sprintf("%s %s", keyStyle.Render("tab"), descStyle.Render("focus")),
		fmt.Sprintf("%s %s", keyStyle.Render("b"), descStyle.Render("boards")),
		fmt.Sprintf("%s %s", keyStyle.Render("r"), descStyle.Render("refresh")),
		fmt.Sprintf("%s %s", keyStyle.Render("q"), descStyle.Render("quit")),
	}
	right := strings.Join(hints, sep)

	// Compose full bar
	barStyle := lipgloss.NewStyle().
		Background(draculaLine).
		Width(width)

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := width - leftW - rightW
	if gap < 1 {
		gap = 1
	}

	return barStyle.Render(left + strings.Repeat(" ", gap) + right)
}
