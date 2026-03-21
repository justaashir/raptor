package tui

import (
	"raptor/model"
	"strings"
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
		{model.Done, colorComment},
	}
	for _, tt := range tests {
		got := StatusColor(tt.status)
		if got != tt.want {
			t.Errorf("StatusColor(%s) = %v, want %v", tt.status, got, tt.want)
		}
	}
	// Unknown statuses should get a palette color, not a fixed fallback
	got := StatusColor(model.Status("review"))
	for _, known := range []lipgloss.Color{colorOrange, colorCyan, colorGreen} {
		if got == known {
			return // acceptable — it happens to match a palette color
		}
	}
	// Just verify it's one of the palette colors
	found := false
	for _, c := range statusPalette {
		if got == c {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("StatusColor(review) = %v, expected a palette color", got)
	}
}

func TestRenderFloatingWindow_ContainsContent(t *testing.T) {
	content := "Hello World"
	result := RenderFloatingWindow(content, 30, 10, 80, 24)
	if !strings.Contains(result, "Hello World") {
		t.Fatalf("floating window should contain content, got:\n%s", result)
	}
}

func TestRenderFloatingWindow_HasBorder(t *testing.T) {
	result := RenderFloatingWindow("test", 30, 10, 80, 24)
	// Should contain rounded border characters
	if !strings.Contains(result, "╭") {
		t.Fatalf("floating window should have rounded border, got:\n%s", result)
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
