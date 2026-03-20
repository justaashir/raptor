package tui

import (
	"fmt"
	"io"
	"raptor/model"

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

	icon := StatusIcon(t.Status)
	status := FormatStatus(t.Status)
	assignee := "·"
	if t.Assignee != "" {
		assignee = "@" + t.Assignee
	}
	age := FormatAge(t.CreatedAt)

	// icon(2) + STATUS(7) + ID(9) + ASSIGNEE(10) + AGE(5) + spaces = ~37
	fixedW := 37
	titleW := d.width - fixedW
	if titleW < 4 {
		titleW = 4
	}
	title := truncate(t.Title, titleW)

	isSelected := index == m.Index()

	if isSelected {
		// Selected row — purple highlight like beads_viewer
		cursor := lipgloss.NewStyle().Foreground(draculaPurple).Render("▸")
		row := lipgloss.NewStyle().
			Background(draculaLine).
			Foreground(draculaFg).
			Bold(true).
			MaxWidth(d.width).
			Render(fmt.Sprintf("%s %s %-7s %-9s %-10s %-5s %s",
				cursor, icon, status, t.ID, assignee, age, title))
		fmt.Fprint(w, row)
		return
	}

	// Normal row with colored columns
	row := fmt.Sprintf("%s %s %s %s %s %s",
		lipgloss.NewStyle().Foreground(StatusColor(t.Status)).Render(icon),
		lipgloss.NewStyle().Foreground(StatusColor(t.Status)).Width(7).Render(status),
		lipgloss.NewStyle().Foreground(draculaComment).Width(9).Render(t.ID),
		lipgloss.NewStyle().Foreground(draculaPurple).Width(10).Render(assignee),
		lipgloss.NewStyle().Foreground(draculaComment).Width(5).Render(age),
		lipgloss.NewStyle().Foreground(draculaFg).MaxWidth(titleW).Render(title),
	)
	fmt.Fprint(w, row)
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
