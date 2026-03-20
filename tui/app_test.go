package tui

import (
	"raptor/model"
	"testing"
	"time"
)

func sampleTickets() []model.Ticket {
	now := time.Now()
	return []model.Ticket{
		{ID: "aaaa1111", Title: "Task A", Status: model.Todo, Assignee: "tom", CreatedAt: now.Add(-2 * 24 * time.Hour), UpdatedAt: now},
		{ID: "bbbb2222", Title: "Task B", Status: model.InProgress, Assignee: "alice", CreatedAt: now.Add(-5 * time.Hour), UpdatedAt: now},
		{ID: "cccc3333", Title: "Task C", Status: model.Done, CreatedAt: now.Add(-7 * 24 * time.Hour), UpdatedAt: now},
		{ID: "dddd4444", Title: "Task D", Status: model.Todo, CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now},
	}
}

func TestApp_NewApp_CreatesListAndDetailPanes(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	if app.listPane == nil {
		t.Fatal("listPane should not be nil")
	}
	if app.detailPane == nil {
		t.Fatal("detailPane should not be nil")
	}
}

func TestApp_SetTickets_PopulatesListAndDetail(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	app.width = 120
	app.height = 40
	app.initPanes()
	app.SetTickets(sampleTickets())

	if app.listPane.RowCount() != 4 {
		t.Fatalf("expected 4 rows, got %d", app.listPane.RowCount())
	}

	selected := app.listPane.SelectedTicket()
	if selected == nil || selected.ID != "aaaa1111" {
		t.Fatal("first ticket should be selected after SetTickets")
	}
}

func TestApp_FocusedPane_DefaultIsList(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	if app.focused != focusList {
		t.Fatalf("expected focusList, got %d", app.focused)
	}
}

func TestApp_ToggleFocus_SwitchesBetweenPanes(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")

	app.toggleFocus()
	if app.focused != focusDetail {
		t.Fatalf("expected focusDetail after toggle, got %d", app.focused)
	}

	app.toggleFocus()
	if app.focused != focusList {
		t.Fatalf("expected focusList after second toggle, got %d", app.focused)
	}
}

func TestApp_SelectedTicket_ReturnsFromList(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	app.width = 120
	app.height = 40
	app.initPanes()
	app.SetTickets(sampleTickets())

	selected := app.SelectedTicket()
	if selected == nil || selected.ID != "aaaa1111" {
		t.Fatal("SelectedTicket should return first ticket")
	}
}

func TestApp_SelectedTicket_NilWhenEmpty(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	if app.SelectedTicket() != nil {
		t.Fatal("SelectedTicket should be nil when no tickets")
	}
}

func TestApp_AllTickets_StoredForStatusBar(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	app.width = 120
	app.height = 40
	app.initPanes()
	app.SetTickets(sampleTickets())

	if len(app.tickets) != 4 {
		t.Fatalf("expected 4 tickets stored, got %d", len(app.tickets))
	}
}
