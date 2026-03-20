package tui

import (
	"raptor/model"
	"testing"
)

func TestColumn_CursorMovement(t *testing.T) {
	tickets := []model.Ticket{
		{ID: "a", Title: "First", Status: model.Todo},
		{ID: "b", Title: "Second", Status: model.Todo},
		{ID: "c", Title: "Third", Status: model.Todo},
	}
	col := NewColumn("Todo", model.Todo, tickets)

	if col.Cursor() != 0 {
		t.Fatalf("expected cursor at 0, got %d", col.Cursor())
	}

	col.MoveDown()
	if col.Cursor() != 1 {
		t.Fatalf("expected cursor at 1, got %d", col.Cursor())
	}

	col.MoveDown()
	col.MoveDown() // should clamp at last item
	if col.Cursor() != 2 {
		t.Fatalf("expected cursor at 2, got %d", col.Cursor())
	}

	col.MoveUp()
	if col.Cursor() != 1 {
		t.Fatalf("expected cursor at 1, got %d", col.Cursor())
	}

	col.MoveUp()
	col.MoveUp() // should clamp at 0
	if col.Cursor() != 0 {
		t.Fatalf("expected cursor at 0, got %d", col.Cursor())
	}
}

func TestColumn_SelectedTicket(t *testing.T) {
	tickets := []model.Ticket{
		{ID: "a", Title: "First"},
		{ID: "b", Title: "Second"},
	}
	col := NewColumn("Todo", model.Todo, tickets)

	selected := col.SelectedTicket()
	if selected == nil || selected.ID != "a" {
		t.Fatalf("expected first ticket selected, got %v", selected)
	}

	col.MoveDown()
	selected = col.SelectedTicket()
	if selected == nil || selected.ID != "b" {
		t.Fatalf("expected second ticket selected, got %v", selected)
	}
}

func TestColumn_EmptySelectedTicket(t *testing.T) {
	col := NewColumn("Empty", model.Todo, nil)
	if col.SelectedTicket() != nil {
		t.Fatal("expected nil for empty column")
	}
}
