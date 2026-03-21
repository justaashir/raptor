package tui

import (
	"fmt"
	"raptor/model"
	"sort"
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
func RenderStatusBar(tickets []model.Ticket, boardName string, width int) string {
	counts := CountByStatus(tickets)

	// Filter badge
	filterBadge := lipgloss.NewStyle().
		Background(colorPurple).
		Foreground(lipgloss.Color("#282a36")).
		Bold(true).
		Padding(0, 1).
		Render("ALL")

	// Ticket counts — sort statuses for deterministic display
	var statuses []model.Status
	for status := range counts {
		statuses = append(statuses, status)
	}
	sort.Slice(statuses, func(i, j int) bool {
		return string(statuses[i]) < string(statuses[j])
	})
	countParts := []string{}
	for _, status := range statuses {
		count := counts[status]
		icon := StatusIcon(status)
		color := StatusColor(status)
		countParts = append(countParts, lipgloss.NewStyle().Foreground(color).Render(
			fmt.Sprintf("%s%d", icon, count)))
	}
	left := fmt.Sprintf("%s  %s  %d tickets  %s",
		filterBadge,
		lipgloss.NewStyle().Foreground(colorPink).Bold(true).Render(boardName),
		len(tickets),
		strings.Join(countParts, " "),
	)

	// Right side: keybind hints with separators
	keyStyle := lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(colorComment)
	sep := lipgloss.NewStyle().Foreground(colorComment).Render(" │ ")

	hints := []string{
		fmt.Sprintf("%s %s", keyStyle.Render("j/k"), descStyle.Render("nav")),
		fmt.Sprintf("%s %s", keyStyle.Render("/"), descStyle.Render("filter")),
		fmt.Sprintf("%s %s", keyStyle.Render("tab"), descStyle.Render("focus")),
		fmt.Sprintf("%s %s", keyStyle.Render("n"), descStyle.Render("new")),
		fmt.Sprintf("%s %s", keyStyle.Render("w"), descStyle.Render("workspace")),
		fmt.Sprintf("%s %s", keyStyle.Render("b"), descStyle.Render("boards")),
		fmt.Sprintf("%s %s", keyStyle.Render("r"), descStyle.Render("refresh")),
		fmt.Sprintf("%s %s", keyStyle.Render("q"), descStyle.Render("quit")),
	}
	right := strings.Join(hints, sep)

	// Compose full bar
	barStyle := lipgloss.NewStyle().
		Background(colorLine).
		Width(width)

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := width - leftW - rightW
	if gap < 1 {
		gap = 1
	}

	return barStyle.Render(left + strings.Repeat(" ", gap) + right)
}
