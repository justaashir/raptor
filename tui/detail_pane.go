package tui

import (
	"fmt"
	"raptor/model"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// DetailPane wraps a bubbles/viewport for displaying ticket details.
type DetailPane struct {
	viewport viewport.Model
	width    int
	height   int
}

// NewDetailPane creates a new detail pane with the given dimensions.
func NewDetailPane(width, height int) *DetailPane {
	vp := viewport.New(width, height)
	vp.SetContent("No ticket selected")
	return &DetailPane{
		viewport: vp,
		width:    width,
		height:   height,
	}
}

// RenderDetailContent renders ticket detail as a string. Returns placeholder for nil ticket.
func RenderDetailContent(t *model.Ticket, width int) string {
	if t == nil {
		return "No ticket selected"
	}

	title := DetailTitleStyle.Render(t.Title)

	metaLines := fmt.Sprintf(
		"%s %s  %s %s  %s %s",
		DetailMetaKeyStyle.Render("Status"),
		lipgloss.NewStyle().Foreground(StatusColor(t.Status)).Render(FormatStatus(t.Status)),
		DetailMetaKeyStyle.Render("ID"),
		DetailMetaValueStyle.Render(t.ID),
		DetailMetaKeyStyle.Render("Age"),
		DetailMetaValueStyle.Render(FormatAge(t.CreatedAt)),
	)

	if t.Assignee != "" {
		metaLines += fmt.Sprintf("  %s %s",
			DetailMetaKeyStyle.Render("Assignee"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("176")).Render("@"+t.Assignee),
		)
	}

	if t.CreatedBy != "" {
		metaLines += fmt.Sprintf("  %s %s",
			DetailMetaKeyStyle.Render("By"),
			DetailMetaValueStyle.Render(t.CreatedBy),
		)
	}

	dates := fmt.Sprintf(
		"%s %s  %s %s",
		DetailMetaKeyStyle.Render("Created"),
		DetailMetaValueStyle.Render(t.CreatedAt.Format("2006-01-02 15:04")),
		DetailMetaKeyStyle.Render("Updated"),
		DetailMetaValueStyle.Render(t.UpdatedAt.Format("2006-01-02 15:04")),
	)

	var body string
	if t.Content != "" {
		contentWidth := width - 4
		if contentWidth < 20 {
			contentWidth = 20
		}
		r, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(contentWidth),
		)
		if err == nil {
			rendered, err := r.Render(t.Content)
			if err == nil {
				body = rendered
			} else {
				body = t.Content
			}
		} else {
			body = t.Content
		}
	}

	separator := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("─────────────────────────────")

	return title + "\n\n" + metaLines + "\n" + dates + "\n\n" + separator + "\n" + body
}

// SetTicket updates the detail pane with a new ticket.
func (dp *DetailPane) SetTicket(t *model.Ticket) {
	content := RenderDetailContent(t, dp.width)
	dp.viewport.SetContent(content)
	dp.viewport.GotoTop()
}

// Update delegates to the viewport Update.
func (dp *DetailPane) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	dp.viewport, cmd = dp.viewport.Update(msg)
	return cmd
}

// View renders the viewport.
func (dp *DetailPane) View() string {
	return dp.viewport.View()
}

// SetSize updates the pane dimensions.
func (dp *DetailPane) SetSize(width, height int) {
	dp.width = width
	dp.height = height
	dp.viewport.Width = width
	dp.viewport.Height = height
}
