package tui

import (
	"fmt"
	"raptor/model"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
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

func sp(s string) *string { return &s }
func bp(b bool) *bool     { return &b }
func up(u uint) *uint     { return &u }

// raptorStyle builds a glamour StyleConfig matching the Dracula palette,
// modelled after beads_viewer's buildStyleFromTheme.
func raptorStyle() ansi.StyleConfig {
	fg := "#f8f8f2"
	purple := "#bd93f9"
	pink := "#ff79c6"
	cyan := "#8be9fd"
	green := "#50fa7b"
	orange := "#ffb86c"
	yellow := "#f1fa8c"
	comment := "#6272a4"
	tableBg := "#383a4a"

	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: sp(fg)},
			Margin:         up(0),
		},
		Paragraph: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: sp(fg)},
			Margin:         up(0),
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: sp(purple), Bold: bp(true)},
			Margin:         up(0),
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: sp(purple), Bold: bp(true), BlockSuffix: "\n"},
			Margin:         up(0),
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: sp(pink), Bold: bp(true)},
			Margin:         up(0),
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: sp(cyan), Bold: bp(true)},
			Margin:         up(0),
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: sp(orange), Bold: bp(true)},
			Margin:         up(0),
		},
		Strong: ansi.StylePrimitive{
			Color: sp(orange),
			Bold:  bp(true),
		},
		Emph: ansi.StylePrimitive{
			Color:  sp(yellow),
			Italic: bp(true),
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: sp(comment), Italic: bp(true)},
			Indent:         up(2),
			Margin:         up(0),
		},
		List: ansi.StyleList{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: sp(fg)},
				Margin:         up(0),
			},
			LevelIndent: 2,
		},
		Item: ansi.StylePrimitive{
			BlockPrefix: "• ",
		},
		Link: ansi.StylePrimitive{
			Color:     sp(cyan),
			Underline: bp(true),
		},
		LinkText: ansi.StylePrimitive{
			Color: sp(purple),
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: sp(green)},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: sp(green)},
				Margin:         up(0),
			},
			Chroma: &ansi.Chroma{
				Text:    ansi.StylePrimitive{Color: sp(fg)},
				Keyword: ansi.StylePrimitive{Color: sp(purple)},
				Name:    ansi.StylePrimitive{Color: sp(cyan)},
				Comment: ansi.StylePrimitive{Color: sp(comment), Italic: bp(true)},
				LiteralString: ansi.StylePrimitive{Color: sp(yellow)},
				LiteralNumber: ansi.StylePrimitive{Color: sp(purple)},
			},
		},
		HorizontalRule: ansi.StylePrimitive{
			Color:  sp(comment),
			Format: "─────────────────────────────────────────",
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color:           sp(fg),
					BackgroundColor: sp(tableBg),
				},
				Margin: up(0),
			},
			CenterSeparator: sp("┼"),
			ColumnSeparator: sp("│"),
			RowSeparator:    sp("─"),
		},
	}
}

// newGlamourRenderer creates a cached glamour renderer with Dracula-themed styling.
func newGlamourRenderer(width int) *glamour.TermRenderer {
	contentWidth := width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(raptorStyle()),
		glamour.WithWordWrap(contentWidth),
		glamour.WithEmoji(),
	)
	if err != nil {
		return nil
	}
	return r
}

// RenderDetailContent renders ticket detail as markdown, then renders through glamour.
// Accepts an optional cached glamour renderer for performance.
func RenderDetailContent(t *model.Ticket, width int, renderer ...*glamour.TermRenderer) string {
	if t == nil {
		return "No ticket selected"
	}

	var r *glamour.TermRenderer
	if len(renderer) > 0 && renderer[0] != nil {
		r = renderer[0]
	} else {
		r = newGlamourRenderer(width)
	}

	// Build everything as markdown, render once through glamour
	var md string

	// Title
	md += fmt.Sprintf("# %s %s\n\n", StatusIcon(t.Status), t.Title)

	// Metadata table (like beads_viewer)
	assignee := "—"
	if t.Assignee != "" {
		assignee = "@" + t.Assignee
	}
	createdBy := "—"
	if t.CreatedBy != "" {
		createdBy = t.CreatedBy
	}

	md += "| ID | Status | Assignee | Age | Created By |\n"
	md += "|---|---|---|---|---|\n"
	md += fmt.Sprintf("| **%s** | **%s** | %s | %s | %s |\n\n",
		t.ID,
		FormatStatus(t.Status),
		assignee,
		FormatAge(t.CreatedAt),
		createdBy,
	)

	// Content / description
	if t.Content != "" {
		md += t.Content + "\n"
	}

	if r != nil {
		rendered, err := r.Render(md)
		if err == nil {
			return rendered
		}
	}
	return md
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

// SetSize updates the pane dimensions and recreates the glamour renderer only on width change.
func (dp *DetailPane) SetSize(width, height int) {
	if dp.width != width {
		dp.renderer = newGlamourRenderer(width)
	}
	dp.width = width
	dp.height = height
	dp.viewport.Width = width
	dp.viewport.Height = height
}
