package tui

import (
	"raptor/model"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestStatusColor_ReturnsOrangeForTodo(t *testing.T) {
	got := StatusColor(model.Todo)
	want := lipgloss.Color("#ffb86c")
	if got != want {
		t.Fatalf("StatusColor(Todo) = %v, want %v", got, want)
	}
}

func TestStatusColor_ReturnsCyanForInProgress(t *testing.T) {
	got := StatusColor(model.InProgress)
	want := lipgloss.Color("#8be9fd")
	if got != want {
		t.Fatalf("StatusColor(InProgress) = %v, want %v", got, want)
	}
}

func TestStatusColor_ReturnsGreenForDone(t *testing.T) {
	got := StatusColor(model.Done)
	want := lipgloss.Color("#50fa7b")
	if got != want {
		t.Fatalf("StatusColor(Done) = %v, want %v", got, want)
	}
}

func TestStatusColor_ReturnsRedForClosed(t *testing.T) {
	got := StatusColor(model.Closed)
	want := lipgloss.Color("#ff5555")
	if got != want {
		t.Fatalf("StatusColor(Closed) = %v, want %v", got, want)
	}
}

func TestStatusColor_ReturnsCommentForUnknown(t *testing.T) {
	got := StatusColor(model.Status("unknown"))
	want := lipgloss.Color("#6272a4")
	if got != want {
		t.Fatalf("StatusColor(unknown) = %v, want %v", got, want)
	}
}

func TestStatusIcon_ReturnsCorrectIcons(t *testing.T) {
	tests := []struct {
		status model.Status
		icon   string
	}{
		{model.Todo, "📋"},
		{model.InProgress, "🔧"},
		{model.Done, "✅"},
		{model.Closed, "🔒"},
	}
	for _, tt := range tests {
		got := StatusIcon(tt.status)
		if got != tt.icon {
			t.Errorf("StatusIcon(%s) = %q, want %q", tt.status, got, tt.icon)
		}
	}
}
