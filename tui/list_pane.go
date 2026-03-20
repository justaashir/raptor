package tui

import (
	"fmt"
	"io"
	"raptor/model"
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
	case model.Closed:
		return "CLOSED"
	default:
		return string(s)
	}
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
func (d ticketDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	ti, ok := listItem.(TicketItem)
	if !ok {
		return
	}
	t := ti.ticket

	status := padRight(FormatStatus(t.Status), 8)
	id := padRight(t.ID, 10)
	assignee := "·"
	if t.Assignee != "" {
		assignee = "@" + t.Assignee
	}
	assignee = padRight(assignee, 10)
	age := padRight(FormatAge(t.CreatedAt), 5)

	// Prefix: icon(2) + star(2) + spaces(2) = 6, then columns
	fixedW := 6 + 8 + 10 + 10 + 5
	titleW := d.width - fixedW
	if titleW < 3 {
		titleW = 3
	}
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

// padRight pads s with spaces to width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// truncate cuts a string to maxLen, adding "..." if it was longer.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
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

// BuildRow converts a ticket into a string slice (for test compatibility).
func (lp *ListPane) BuildRow(t model.Ticket) []string {
	assignee := "--"
	if t.Assignee != "" {
		assignee = "@" + t.Assignee
	}
	return []string{
		FormatStatus(t.Status),
		t.ID,
		assignee,
		FormatAge(t.CreatedAt),
		t.Title,
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
	return ColumnHeaderStyle.Width(lp.width).Render(
		fmt.Sprintf("    %-8s %-10s %-10s %-5s %s",
			"STATUS", "ID", "ASSIGNEE", "AGE", "TITLE"))
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
