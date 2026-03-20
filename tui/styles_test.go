package tui

import (
	"raptor/model"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestStatusColor_ReturnsCorrectColors(t *testing.T) {
	tests := []struct {
		status model.Status
		want   lipgloss.Color
	}{
		{model.Todo, colorOrange},
		{model.InProgress, colorCyan},
		{model.Done, colorGreen},
		{model.Status("unknown"), colorComment},
	}
	for _, tt := range tests {
		got := StatusColor(tt.status)
		if got != tt.want {
			t.Errorf("StatusColor(%s) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestStatusIcon_ReturnsCorrectIcons(t *testing.T) {
	tests := []struct {
		status model.Status
		icon   string
	}{
		{model.Todo, "📋"},
		{model.InProgress, "⚡"},
		{model.Done, "✅"},
	}
	for _, tt := range tests {
		got := StatusIcon(tt.status)
		if got != tt.icon {
			t.Errorf("StatusIcon(%s) = %q, want %q", tt.status, got, tt.icon)
		}
	}
}
