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

func TestOverlay_ContainsContent(t *testing.T) {
	bg := strings.Repeat("BACKGROUND LINE\n", 24)
	content := "Hello World"
	result := overlayOnBackground(content, 30, 10, bg, 80, 24)
	if !strings.Contains(result, "Hello World") {
		t.Fatalf("overlay should contain content, got:\n%s", result)
	}
}

func TestOverlay_HasBorder(t *testing.T) {
	bg := strings.Repeat("BACKGROUND LINE\n", 24)
	result := overlayOnBackground("test", 30, 10, bg, 80, 24)
	if !strings.Contains(result, "╭") {
		t.Fatalf("overlay should have rounded border, got:\n%s", result)
	}
}

func TestOverlay_PreservesBackground(t *testing.T) {
	bg := strings.Repeat("BACKGROUND LINE HERE\n", 24)
	result := overlayOnBackground("tiny", 20, 5, bg, 80, 24)
	// Background lines outside the overlay area should still be present
	if !strings.Contains(result, "BACKGROUND") {
		t.Fatalf("overlay should preserve background lines, got:\n%s", result)
	}
}

func TestOverlay_NoAnsiArtifacts(t *testing.T) {
	bg := strings.Repeat("BACKGROUND LINE HERE AND MORE TEXT PADDING\n", 24)
	result := overlayOnBackground("test", 20, 3, bg, 80, 24)
	// Should not contain broken ANSI sequences
	if strings.Contains(result, ";163m") || strings.Contains(result, ";113") {
		t.Fatalf("overlay should not have broken ANSI sequences, got:\n%s", result)
	}
	// Should have background content both above/below and alongside the overlay
	lines := strings.Split(result, "\n")
	// First line should be fully background (above overlay)
	if !strings.Contains(lines[0], "BACKGROUND") {
		t.Fatalf("first line should be background, got: %q", lines[0])
	}
	// Last line should be fully background (below overlay)
	if !strings.Contains(lines[len(lines)-1], "BACKGROUND") {
		t.Fatalf("last line should be background, got: %q", lines[len(lines)-1])
	}
}

func TestStatusStar(t *testing.T) {
	// Done status should return spaces (no star)
	got := StatusStar(model.Done)
	if got != "  " {
		t.Errorf("StatusStar(Done) = %q, want %q", got, "  ")
	}

	// Other statuses should return star
	for _, s := range []model.Status{model.Todo, model.InProgress, model.Status("review")} {
		got := StatusStar(s)
		if got != "⭐" {
			t.Errorf("StatusStar(%s) = %q, want %q", s, got, "⭐")
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
