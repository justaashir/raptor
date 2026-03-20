package tui

import (
	"raptor/model"

	"github.com/charmbracelet/lipgloss"
)

// Status colors
func StatusColor(s model.Status) lipgloss.Color {
	switch s {
	case model.Todo:
		return lipgloss.Color("208")
	case model.InProgress:
		return lipgloss.Color("81")
	case model.Done:
		return lipgloss.Color("114")
	case model.Closed:
		return lipgloss.Color("203")
	default:
		return lipgloss.Color("240")
	}
}

// Pane styles
var (
	FocusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))

	UnfocusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))

	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	DetailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("176"))

	DetailMetaKeyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	DetailMetaValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))
)
