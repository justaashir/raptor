package tui

import (
	"fmt"
	"io"
	"raptor/model"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FormatStatus returns an abbreviated status string for display.
func FormatStatus(s model.Status) string {
	switch s {
	case model.Todo:
		return "TODO"
	case model.InProgress:
		return "IN_PROG"
	case model.Done:
		return "DONE"
	default:
		return string(s)
	}
}

// sortTicketsDoneLast returns a copy with done tickets at the bottom,
// preserving relative order within each group.
func sortTicketsDoneLast(tickets []model.Ticket) []model.Ticket {
	result := make([]model.Ticket, len(tickets))
	copy(result, tickets)
	sort.SliceStable(result, func(i, j int) bool {
		iDone := result[i].Status == model.Done
		jDone := result[j].Status == model.Done
		if iDone != jDone {
			return !iDone
		}
		return false
	})
	return result
}

// TicketItem wraps a model.Ticket to implement list.Item and list.DefaultItem.
type TicketItem struct {
	ticket model.Ticket
}

func (i TicketItem) FilterValue() string { return i.ticket.Title }
func (i TicketItem) Title() string       { return i.ticket.Title }
func (i TicketItem) Description() string { return i.ticket.ID }

// ticketDelegate renders each ticket as a single-line row with columns.
type ticketDelegate struct {
	width int
}

func (d ticketDelegate) Height() int                               { return 1 }
func (d ticketDelegate) Spacing() int                              { return 0 }
func (d ticketDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd   { return nil }
// colWidths computes column widths from available pane width.
// Returns statusW, idW, assigneeW, ageW, titleW.
// Each width includes a 2-char trailing gap for spacing.
func colWidths(total int) (int, int, int, int, int) {
	const prefix = 5    // icon(2) + star(2) + space(1)
	const statusW = 10  // "IN_PROG" (7) + 3 gap
	const idW = 12      // 8-char short ID + 4 gap
	const ageW = 5      // "2h"/"3d" + 3 gap
	fixed := prefix + statusW + idW + ageW
	remaining := total - fixed
	// Split remaining between assignee and title
	assigneeW := 14
	if assigneeW > remaining/2 {
		assigneeW = remaining / 2
	}
	if assigneeW < 4 {
		assigneeW = 4
	}
	titleW := remaining - assigneeW
	if titleW < 3 {
		titleW = 3
	}
	return statusW, idW, assigneeW, ageW, titleW
}

func (d ticketDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	ti, ok := listItem.(TicketItem)
	if !ok {
		return
	}
	t := ti.ticket

	statusW, idW, assigneeW, ageW, titleW := colWidths(d.width)

	status := padRight(FormatStatus(t.Status), statusW)
	id := padRight(t.ID, idW)
	assignee := "—"
	if t.Assignee != "" {
		assignee = "@" + t.Assignee
	}
	assignee = padRight(assignee, assigneeW)
	age := padRight(FormatAge(t.CreatedAt), ageW)
	title := truncate(t.Title, titleW)

	statusColor := StatusColor(t.Status)
	icon := StatusIcon(t.Status)
	star := StatusStar(t.Status)

	if index == m.Index() {
		raw := fmt.Sprintf("%s%s %s%s%s%s%s", icon, star, status, id, assignee, age, title)
		row := lipgloss.NewStyle().
			Background(colorLine).
			Foreground(colorFg).
			Bold(true).
			MaxWidth(d.width).
			Render(raw)
		fmt.Fprint(w, row)
		return
	}

	titleStyle := lipgloss.NewStyle().Foreground(colorFg)
	if t.Status == model.Todo || t.Status == model.InProgress {
		titleStyle = titleStyle.Bold(true)
	}

	raw := fmt.Sprintf("%s%s %s%s%s%s%s",
		icon, star,
		lipgloss.NewStyle().Foreground(statusColor).Render(status),
		lipgloss.NewStyle().Foreground(colorComment).Render(id),
		lipgloss.NewStyle().Foreground(colorPurple).Render(assignee),
		lipgloss.NewStyle().Foreground(colorComment).Render(age),
		titleStyle.Render(title),
	)
	fmt.Fprint(w, lipgloss.NewStyle().MaxWidth(d.width).Render(raw))
}

// truncateToWidth cuts s to fit within maxW display columns.
func truncateToWidth(s string, maxW int) string {
	w := lipgloss.Width(s)
	if w <= maxW {
		return s
	}
	// Remove runes from the end until it fits
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if lipgloss.Width(string(runes)) <= maxW {
			return string(runes)
		}
	}
	return ""
}

// padRight pads s with spaces to exactly `width` display columns,
// guaranteeing a 2-char trailing gap so columns never touch.
func padRight(s string, width int) string {
	const gap = 2
	maxContent := width - gap
	if maxContent < 1 {
		maxContent = 1
	}
	s = truncateToWidth(s, maxContent)
	w := lipgloss.Width(s)
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}

// truncate cuts s to maxW display columns, adding "..." if truncated.
func truncate(s string, maxW int) string {
	if lipgloss.Width(s) <= maxW {
		return s
	}
	if maxW <= 3 {
		return truncateToWidth(s, maxW)
	}
	return truncateToWidth(s, maxW-3) + "..."
}

// ListPane wraps a bubbles/list for displaying tickets.
type ListPane struct {
	list    list.Model
	tickets []model.Ticket
	width   int
	height  int
}

// NewListPane creates a new list pane with the given dimensions.
func NewListPane(width, height int) *ListPane {
	delegate := ticketDelegate{width: width}
	l := list.New([]list.Item{}, delegate, width, height)
	l.Title = "Tickets"
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.SetShowFilter(true)
	l.SetShowPagination(true)
	l.SetStatusBarItemName("ticket", "tickets")

	// Style the list
	l.Styles.NoItems = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(1, 2)

	return &ListPane{
		list:   l,
		width:  width,
		height: height,
	}
}

// SetTickets updates the list with new ticket data, preserving selection by ID.
func (lp *ListPane) SetTickets(tickets []model.Ticket) {
	// Remember current selection
	var selectedID string
	if old := lp.SelectedTicket(); old != nil {
		selectedID = old.ID
	}

	tickets = sortTicketsDoneLast(tickets)
	lp.tickets = tickets

	items := make([]list.Item, len(tickets))
	for i, t := range tickets {
		items[i] = TicketItem{ticket: t}
	}
	lp.list.SetItems(items)

	// Restore selection by ID
	if selectedID != "" {
		for i, t := range tickets {
			if t.ID == selectedID {
				lp.list.Select(i)
				return
			}
		}
	}
}

// RowCount returns the number of tickets.
func (lp *ListPane) RowCount() int {
	return len(lp.tickets)
}

// SelectedTicket returns the currently selected ticket, or nil if empty.
func (lp *ListPane) SelectedTicket() *model.Ticket {
	if len(lp.tickets) == 0 {
		return nil
	}
	item := lp.list.SelectedItem()
	if item == nil {
		return nil
	}
	ti, ok := item.(TicketItem)
	if !ok {
		return nil
	}
	return &ti.ticket
}

// SetCursor sets the list selection position.
func (lp *ListPane) SetCursor(n int) {
	lp.list.Select(n)
}

// Cursor returns the current index.
func (lp *ListPane) Cursor() int {
	return lp.list.Index()
}

// Filtering returns true if the list is currently in filter mode.
func (lp *ListPane) Filtering() bool {
	return lp.list.FilterState() == list.Filtering
}

// Update delegates to the bubbles list Update.
func (lp *ListPane) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	lp.list, cmd = lp.list.Update(msg)
	return cmd
}

// ColumnHeader returns the styled column header string.
func (lp *ListPane) ColumnHeader() string {
	statusW, idW, assigneeW, ageW, _ := colWidths(lp.width)
	header := fmt.Sprintf("    %s%s%s%s%s",
		padRight("STATUS", statusW),
		padRight("ID", idW),
		padRight("ASSIGNEE", assigneeW),
		padRight("AGE", ageW),
		"TITLE")
	return ColumnHeaderStyle.Width(lp.width).Render(header)
}

// View renders the list.
func (lp *ListPane) View() string {
	return lp.list.View()
}

// SetSize updates the pane dimensions.
func (lp *ListPane) SetSize(width, height int) {
	lp.width = width
	lp.height = height
	lp.list.SetWidth(width)
	lp.list.SetHeight(height)
	lp.list.SetDelegate(ticketDelegate{width: width})
}
