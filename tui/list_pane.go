package tui

import (
	"raptor/model"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
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

// ListPane wraps a bubbles/table for displaying tickets.
type ListPane struct {
	table   table.Model
	tickets []model.Ticket
	width   int
	height  int
}

// NewListPane creates a new list pane with the given dimensions.
func NewListPane(width, height int) *ListPane {
	cols := listColumns(width)

	// Custom KeyMap that doesn't conflict with app-level keybinds
	km := table.KeyMap{
		LineUp:       key.NewBinding(key.WithKeys("up", "k")),
		LineDown:     key.NewBinding(key.WithKeys("down", "j")),
		PageUp:       key.NewBinding(key.WithKeys("pgup")),
		PageDown:     key.NewBinding(key.WithKeys("pgdown")),
		HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u")),
		HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d")),
		GotoTop:      key.NewBinding(key.WithKeys("home")),
		GotoBottom:   key.NewBinding(key.WithKeys("end")),
	}

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("240"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("62")).
		Bold(true)

	t := table.New(
		table.WithColumns(cols),
		table.WithHeight(height),
		table.WithWidth(width),
		table.WithFocused(true),
		table.WithKeyMap(km),
		table.WithStyles(s),
	)

	return &ListPane{
		table:  t,
		width:  width,
		height: height,
	}
}

func listColumns(width int) []table.Column {
	// STATUS(7) + ID(8) + ASSIGNEE(8) + AGE(4) + gaps = ~35, rest is TITLE
	statusW := 7
	idW := 8
	assigneeW := 8
	ageW := 4
	fixedW := statusW + idW + assigneeW + ageW + 8 // padding
	titleW := width - fixedW
	if titleW < 10 {
		titleW = 10
	}
	return []table.Column{
		{Title: "STATUS", Width: statusW},
		{Title: "ID", Width: idW},
		{Title: "ASSIGNEE", Width: assigneeW},
		{Title: "AGE", Width: ageW},
		{Title: "TITLE", Width: titleW},
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

	rows := make([]table.Row, len(tickets))
	for i, t := range tickets {
		rows[i] = lp.BuildRow(t)
	}
	lp.table.SetRows(rows)

	// Restore selection by ID
	if selectedID != "" {
		for i, t := range tickets {
			if t.ID == selectedID {
				lp.table.SetCursor(i)
				return
			}
		}
	}
	// If not found, clamp cursor
	if lp.table.Cursor() >= len(tickets) && len(tickets) > 0 {
		lp.table.SetCursor(len(tickets) - 1)
	}
}

// BuildRow converts a ticket into a table row.
func (lp *ListPane) BuildRow(t model.Ticket) table.Row {
	assignee := "--"
	if t.Assignee != "" {
		assignee = "@" + t.Assignee
	}
	return table.Row{
		FormatStatus(t.Status),
		t.ID,
		assignee,
		FormatAge(t.CreatedAt),
		t.Title,
	}
}

// RowCount returns the number of rows in the table.
func (lp *ListPane) RowCount() int {
	return len(lp.tickets)
}

// SelectedTicket returns the currently selected ticket, or nil if empty.
func (lp *ListPane) SelectedTicket() *model.Ticket {
	if len(lp.tickets) == 0 {
		return nil
	}
	idx := lp.table.Cursor()
	if idx >= len(lp.tickets) {
		return nil
	}
	return &lp.tickets[idx]
}

// SetCursor sets the table cursor position.
func (lp *ListPane) SetCursor(n int) {
	lp.table.SetCursor(n)
}

// Cursor returns the current cursor position.
func (lp *ListPane) Cursor() int {
	return lp.table.Cursor()
}

// Focus gives focus to the table.
func (lp *ListPane) Focus() {
	lp.table.Focus()
}

// Blur removes focus from the table.
func (lp *ListPane) Blur() {
	lp.table.Blur()
}

// Focused returns whether the table is focused.
func (lp *ListPane) Focused() bool {
	return lp.table.Focused()
}

// Update delegates to the bubbles table Update.
func (lp *ListPane) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	lp.table, cmd = lp.table.Update(msg)
	return cmd
}

// View renders the table.
func (lp *ListPane) View() string {
	return lp.table.View()
}

// SetSize updates the pane dimensions.
func (lp *ListPane) SetSize(width, height int) {
	lp.width = width
	lp.height = height
	lp.table.SetWidth(width)
	lp.table.SetHeight(height)
	lp.table.SetColumns(listColumns(width))
}
