package tui

import (
	"fmt"
	"raptor/model"

	"github.com/charmbracelet/lipgloss"
)

type Column struct {
	title   string
	status  model.Status
	tickets []model.Ticket
	cursor  int
	focused bool
}

func NewColumn(title string, status model.Status, tickets []model.Ticket) *Column {
	return &Column{
		title:   title,
		status:  status,
		tickets: tickets,
	}
}

func (c *Column) Cursor() int            { return c.cursor }
func (c *Column) SetFocused(f bool)       { c.focused = f }
func (c *Column) Status() model.Status    { return c.status }
func (c *Column) Tickets() []model.Ticket { return c.tickets }

func (c *Column) SetTickets(tickets []model.Ticket) {
	c.tickets = tickets
	if c.cursor >= len(tickets) {
		c.cursor = max(0, len(tickets)-1)
	}
}

func (c *Column) MoveDown() {
	if c.cursor < len(c.tickets)-1 {
		c.cursor++
	}
}

func (c *Column) MoveUp() {
	if c.cursor > 0 {
		c.cursor--
	}
}

func (c *Column) SelectedTicket() *model.Ticket {
	if len(c.tickets) == 0 {
		return nil
	}
	return &c.tickets[c.cursor]
}

var (
	columnWidth  = 30
	headerStyle  = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	focusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Width(columnWidth)
	unfocusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Width(columnWidth)
	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("62")).
				Padding(0, 1).
				Width(columnWidth - 2)
	normalItemStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Width(columnWidth - 2)
)

func (c *Column) View() string {
	header := headerStyle.Render(fmt.Sprintf("%s (%d)", c.title, len(c.tickets)))

	var items string
	for i, t := range c.tickets {
		title := t.Title
		if len(title) > columnWidth-4 {
			title = title[:columnWidth-7] + "..."
		}
		if i == c.cursor && c.focused {
			items += selectedItemStyle.Render(title) + "\n"
		} else {
			items += normalItemStyle.Render(title) + "\n"
		}
	}

	content := header + "\n" + items

	if c.focused {
		return focusedStyle.Render(content)
	}
	return unfocusedStyle.Render(content)
}
