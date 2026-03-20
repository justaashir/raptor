package tui

import (
	"raptor/model"
	"strings"
	"testing"
	"time"
)

func TestNewDetailPane_Creates(t *testing.T) {
	dp := NewDetailPane(60, 20)
	if dp == nil {
		t.Fatal("NewDetailPane returned nil")
	}
}

func TestDetailPane_RenderDetail_FullTicket(t *testing.T) {
	now := time.Now()
	ticket := &model.Ticket{
		ID:        "a1b2c3d4",
		Title:     "Fix login redirect bug",
		Content:   "After login, users go to wrong page.",
		Status:    model.Todo,
		Assignee:  "tom",
		CreatedBy: "alice",
		CreatedAt: now.Add(-2 * 24 * time.Hour),
		UpdatedAt: now.Add(-1 * time.Hour),
	}

	content := RenderDetailContent(ticket, 60)

	if !strings.Contains(content, "Fix login redirect bug") {
		t.Fatal("content should contain title")
	}
	if !strings.Contains(content, "a1b2c3d4") {
		t.Fatal("content should contain ID")
	}
	if !strings.Contains(content, "TODO") {
		t.Fatal("content should contain status")
	}
	if !strings.Contains(content, "tom") {
		t.Fatal("content should contain assignee")
	}
	if !strings.Contains(content, "alice") {
		t.Fatal("content should contain creator")
	}
	if !strings.Contains(content, "After login") {
		t.Fatal("content should contain markdown body")
	}
}

func TestDetailPane_RenderDetail_EmptyContent(t *testing.T) {
	ticket := &model.Ticket{
		ID:     "a1b2c3d4",
		Title:  "No content ticket",
		Status: model.Done,
	}

	content := RenderDetailContent(ticket, 60)

	if !strings.Contains(content, "No content ticket") {
		t.Fatal("content should contain title")
	}
	// Should still render without error even with empty content
	if content == "" {
		t.Fatal("content should not be empty")
	}
}

func TestDetailPane_RenderDetail_NilTicket(t *testing.T) {
	content := RenderDetailContent(nil, 60)

	if !strings.Contains(content, "No ticket selected") {
		t.Fatalf("nil ticket should show placeholder, got %q", content)
	}
}

func TestDetailPane_SetTicket_UpdatesContent(t *testing.T) {
	dp := NewDetailPane(60, 20)
	ticket := &model.Ticket{
		ID:     "a1b2c3d4",
		Title:  "Test ticket",
		Status: model.Todo,
	}

	dp.SetTicket(ticket)
	view := dp.View()

	if !strings.Contains(view, "Test ticket") {
		t.Fatal("view should contain ticket title after SetTicket")
	}
}
