package tui

import (
	"raptor/model"
	"testing"
)

func sampleTickets() []model.Ticket {
	return []model.Ticket{
		{ID: "a", Title: "Task A", Status: model.Todo},
		{ID: "b", Title: "Task B", Status: model.InProgress},
		{ID: "c", Title: "Task C", Status: model.Done},
		{ID: "d", Title: "Task D", Status: model.Todo},
	}
}

func TestApp_InitialState(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	app.SetTickets(sampleTickets())

	if app.ActiveColumn() != 0 {
		t.Fatalf("expected active column 0, got %d", app.ActiveColumn())
	}

	col := app.Columns()[0]
	if len(col.Tickets()) != 2 {
		t.Fatalf("expected 2 todo tickets, got %d", len(col.Tickets()))
	}
}

func TestApp_SwitchColumns(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	app.SetTickets(sampleTickets())

	app.MoveRight()
	if app.ActiveColumn() != 1 {
		t.Fatalf("expected column 1, got %d", app.ActiveColumn())
	}

	app.MoveRight()
	if app.ActiveColumn() != 2 {
		t.Fatalf("expected column 2, got %d", app.ActiveColumn())
	}

	app.MoveRight() // clamp
	if app.ActiveColumn() != 2 {
		t.Fatalf("expected column 2 (clamped), got %d", app.ActiveColumn())
	}

	app.MoveLeft()
	if app.ActiveColumn() != 1 {
		t.Fatalf("expected column 1, got %d", app.ActiveColumn())
	}
}

func TestApp_SelectedTicket(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	app.SetTickets(sampleTickets())

	selected := app.SelectedTicket()
	if selected == nil || selected.ID != "a" {
		t.Fatalf("expected ticket a, got %v", selected)
	}

	app.MoveRight() // in_progress column
	selected = app.SelectedTicket()
	if selected == nil || selected.ID != "b" {
		t.Fatalf("expected ticket b, got %v", selected)
	}
}

func TestApp_DynamicColumns(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	// Default 3 columns
	if len(app.Columns()) != 3 {
		t.Fatalf("expected 3 default columns, got %d", len(app.Columns()))
	}

	// Simulate board info with custom statuses
	app.statuses = []string{"backlog", "active", "review", "shipped"}
	app.columns = makeColumns(app.statuses)
	app.columns[0].SetFocused(true)

	if len(app.Columns()) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(app.Columns()))
	}
	if app.Columns()[0].Status() != "backlog" {
		t.Fatalf("expected first column backlog, got %s", app.Columns()[0].Status())
	}
	if app.Columns()[3].Status() != "shipped" {
		t.Fatalf("expected last column shipped, got %s", app.Columns()[3].Status())
	}
}
