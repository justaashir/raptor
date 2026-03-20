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

// RenderStatusBar renders the bottom status bar.
func RenderStatusBar(tickets []model.Ticket, boardName string, focus focusPane, width int) string {
	counts := CountByStatus(tickets)

	// Left side: board info + counts
	todoStyle := lipgloss.NewStyle().Foreground(StatusColor(model.Todo))
	progStyle := lipgloss.NewStyle().Foreground(StatusColor(model.InProgress))
	doneStyle := lipgloss.NewStyle().Foreground(StatusColor(model.Done))
	closedStyle := lipgloss.NewStyle().Foreground(StatusColor(model.Closed))

	filterBadge := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("229")).
		Padding(0, 1).
		Render("ALL")

	left := fmt.Sprintf("%s  %s  %d tickets  %s %s %s %s",
		filterBadge,
		boardName,
		len(tickets),
		todoStyle.Render(fmt.Sprintf("%d todo", counts[model.Todo])),
		progStyle.Render(fmt.Sprintf("%d prog", counts[model.InProgress])),
		doneStyle.Render(fmt.Sprintf("%d done", counts[model.Done])),
		closedStyle.Render(fmt.Sprintf("%d closed", counts[model.Closed])),
	)

	// Right side: keybind hints
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	hints := []string{
		hintStyle.Render("tab") + " focus",
		hintStyle.Render("n") + " new",
		hintStyle.Render("m") + " move",
		hintStyle.Render("e") + " edit",
		hintStyle.Render("x") + " delete",
		hintStyle.Render("q") + " quit",
	}
	right := strings.Join(hints, "  ")

	// Compose full bar
	barStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Width(width)

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := width - leftW - rightW
	if gap < 1 {
		gap = 1
	}

	return barStyle.Render(left + strings.Repeat(" ", gap) + right)
}
