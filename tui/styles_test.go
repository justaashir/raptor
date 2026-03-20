package tui

import (
	"raptor/model"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestStatusColor_ReturnsOrangeForTodo(t *testing.T) {
	got := StatusColor(model.Todo)
	want := lipgloss.Color("208")
	if got != want {
		t.Fatalf("StatusColor(Todo) = %v, want %v", got, want)
	}
}

func TestStatusColor_ReturnsCyanForInProgress(t *testing.T) {
	got := StatusColor(model.InProgress)
	want := lipgloss.Color("81")
	if got != want {
		t.Fatalf("StatusColor(InProgress) = %v, want %v", got, want)
	}
}

func TestStatusColor_ReturnsGreenForDone(t *testing.T) {
	got := StatusColor(model.Done)
	want := lipgloss.Color("114")
	if got != want {
		t.Fatalf("StatusColor(Done) = %v, want %v", got, want)
	}
}

func TestStatusColor_ReturnsRedForClosed(t *testing.T) {
	got := StatusColor(model.Closed)
	want := lipgloss.Color("203")
	if got != want {
		t.Fatalf("StatusColor(Closed) = %v, want %v", got, want)
	}
}

func TestStatusColor_ReturnsGrayForUnknown(t *testing.T) {
	got := StatusColor(model.Status("unknown"))
	want := lipgloss.Color("240")
	if got != want {
		t.Fatalf("StatusColor(unknown) = %v, want %v", got, want)
	}
}
