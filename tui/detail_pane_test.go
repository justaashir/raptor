package tui

import (
	"raptor/model"
	"regexp"
	"strings"
	"testing"
	"time"
)

// stripAnsi removes ANSI escape sequences so glamour output can be substring-matched.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(s string) string { return ansiRe.ReplaceAllString(s, "") }

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

	content := stripAnsi(RenderDetailContent(ticket, 80))

	if !strings.Contains(content, "Fix login redirect bug") {
		t.Fatalf("content should contain title, got:\n%s", content)
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

	content := stripAnsi(RenderDetailContent(ticket, 80))

	if !strings.Contains(content, "No content ticket") {
		t.Fatalf("content should contain title, got:\n%s", content)
	}
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
	dp := NewDetailPane(80, 20)
	ticket := &model.Ticket{
		ID:     "a1b2c3d4",
		Title:  "Test ticket",
		Status: model.Todo,
	}

	dp.SetTicket(ticket)
	view := stripAnsi(dp.View())

	if !strings.Contains(view, "Test ticket") {
		t.Fatalf("view should contain ticket title after SetTicket, got:\n%s", view)
	}
}
