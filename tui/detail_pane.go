package tui

import (
	"fmt"
	"raptor/model"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
)

// DetailPane wraps a bubbles/viewport for displaying ticket details.
type DetailPane struct {
	viewport viewport.Model
	renderer *glamour.TermRenderer
	width    int
	height   int
}

// NewDetailPane creates a new detail pane with the given dimensions.
func NewDetailPane(width, height int) *DetailPane {
	vp := viewport.New(width, height)
	vp.SetContent("No ticket selected")

	renderer := newGlamourRenderer(width)

	return &DetailPane{
		viewport: vp,
		renderer: renderer,
		width:    width,
		height:   height,
	}
}

// newGlamourRenderer creates a cached glamour renderer with proper styling.
func newGlamourRenderer(width int) *glamour.TermRenderer {
	contentWidth := width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(styles.DraculaStyle),
		glamour.WithWordWrap(contentWidth),
		glamour.WithEmoji(),
	)
	if err != nil {
		return nil
	}
	return r
}

// RenderDetailContent renders ticket detail as a string. Returns placeholder for nil ticket.
// Accepts an optional cached glamour renderer for performance.
func RenderDetailContent(t *model.Ticket, width int, renderer ...*glamour.TermRenderer) string {
	if t == nil {
		return "No ticket selected"
	}

	// Title with icon
	icon := lipgloss.NewStyle().Foreground(StatusColor(t.Status)).Render(StatusIcon(t.Status))
	title := fmt.Sprintf("%s %s", icon, DetailTitleStyle.Render(t.Title))

	// Status badge
	statusBadge := lipgloss.NewStyle().
		Background(StatusColor(t.Status)).
		Foreground(lipgloss.Color("#282a36")).
		Bold(true).
		Padding(0, 1).
		Render(FormatStatus(t.Status))

	metaLines := fmt.Sprintf(
		"%s  %s %s  %s %s",
		statusBadge,
		DetailMetaKeyStyle.Render("ID"),
		lipgloss.NewStyle().Foreground(colorCyan).Render(t.ID),
		DetailMetaKeyStyle.Render("Age"),
		lipgloss.NewStyle().Foreground(colorYellow).Render(FormatAge(t.CreatedAt)),
	)

	if t.Assignee != "" {
		metaLines += fmt.Sprintf("  %s %s",
			DetailMetaKeyStyle.Render("Assignee"),
			lipgloss.NewStyle().Foreground(colorPurple).Render("@"+t.Assignee),
		)
	}

	if t.CreatedBy != "" {
		metaLines += fmt.Sprintf("  %s %s",
			DetailMetaKeyStyle.Render("By"),
			lipgloss.NewStyle().Foreground(colorOrange).Render(t.CreatedBy),
		)
	}

	dates := fmt.Sprintf(
		"%s %s",
		DetailMetaKeyStyle.Render("Updated"),
		lipgloss.NewStyle().Foreground(colorComment).Render(t.UpdatedAt.Format("2006-01-02 15:04")),
	)

	var body string
	if t.Content != "" {
		// Use cached renderer if provided, otherwise create one
		var r *glamour.TermRenderer
		if len(renderer) > 0 && renderer[0] != nil {
			r = renderer[0]
		} else {
			r = newGlamourRenderer(width)
		}
		if r != nil {
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
	content := RenderDetailContent(t, dp.width, dp.renderer)
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

// SetSize updates the pane dimensions and recreates the glamour renderer.
func (dp *DetailPane) SetSize(width, height int) {
	dp.width = width
	dp.height = height
	dp.viewport.Width = width
	dp.viewport.Height = height
	dp.renderer = newGlamourRenderer(width)
}
