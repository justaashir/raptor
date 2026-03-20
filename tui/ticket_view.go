package tui

import (
	"fmt"
	"raptor/model"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var (
	detailTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).MarginBottom(1)
	detailMetaStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func RenderTicketDetail(t model.Ticket) string {
	title := detailTitleStyle.Render(t.Title)
	metaStr := fmt.Sprintf(
		"ID: %s  |  Status: %s  |  Created: %s  |  Updated: %s",
		t.ID, t.Status,
		t.CreatedAt.Format("2006-01-02 15:04"),
		t.UpdatedAt.Format("2006-01-02 15:04"),
	)
	if t.CreatedBy != "" {
		metaStr += fmt.Sprintf("  |  By: %s", t.CreatedBy)
	}
	if t.Assignee != "" {
		metaStr += fmt.Sprintf("  |  Assignee: %s", t.Assignee)
	}
	meta := detailMetaStyle.Render(metaStr)

	var content string
	if t.Content != "" {
		rendered, err := glamour.Render(t.Content, "dark")
		if err != nil {
			content = t.Content
		} else {
			content = rendered
		}
	}

	return title + "\n" + meta + "\n\n" + content
}
