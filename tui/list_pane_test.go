package tui

import (
	"raptor/model"
	"testing"
	"time"
)

func TestFormatStatus_Todo(t *testing.T) {
	got := FormatStatus(model.Todo)
	if got != "TODO" {
		t.Fatalf("FormatStatus(Todo) = %q, want %q", got, "TODO")
	}
}

func TestFormatStatus_InProgress(t *testing.T) {
	got := FormatStatus(model.InProgress)
	if got != "IN_PROG" {
		t.Fatalf("FormatStatus(InProgress) = %q, want %q", got, "IN_PROG")
	}
}

func TestFormatStatus_Done(t *testing.T) {
	got := FormatStatus(model.Done)
	if got != "DONE" {
		t.Fatalf("FormatStatus(Done) = %q, want %q", got, "DONE")
	}
}

func TestFormatStatus_Closed(t *testing.T) {
	got := FormatStatus(model.Closed)
	if got != "CLOSED" {
		t.Fatalf("FormatStatus(Closed) = %q, want %q", got, "CLOSED")
	}
}

func testTickets() []model.Ticket {
	now := time.Now()
	return []model.Ticket{
		{ID: "a1b2c3d4", Title: "Fix login bug", Status: model.Todo, Assignee: "tom", CreatedAt: now.Add(-2 * 24 * time.Hour)},
		{ID: "e5f6g7h8", Title: "Add WebSocket reconnect", Status: model.InProgress, Assignee: "alice", CreatedAt: now.Add(-5 * time.Hour)},
		{ID: "i9j0k1l2", Title: "Update API docs", Status: model.Done, CreatedAt: now.Add(-7 * 24 * time.Hour)},
	}
}

func TestNewListPane_CreatesTable(t *testing.T) {
	lp := NewListPane(80, 20)
	if lp == nil {
		t.Fatal("NewListPane returned nil")
	}
}

func TestListPane_SetTickets_PopulatesRows(t *testing.T) {
	lp := NewListPane(80, 20)
	tickets := testTickets()
	lp.SetTickets(tickets)

	if lp.RowCount() != 3 {
		t.Fatalf("RowCount() = %d, want 3", lp.RowCount())
	}
}

func TestListPane_SelectedTicket_ReturnsFirst(t *testing.T) {
	lp := NewListPane(80, 20)
	lp.SetTickets(testTickets())

	selected := lp.SelectedTicket()
	if selected == nil {
		t.Fatal("SelectedTicket() returned nil")
	}
	if selected.ID != "a1b2c3d4" {
		t.Fatalf("SelectedTicket().ID = %q, want %q", selected.ID, "a1b2c3d4")
	}
}

func TestListPane_SelectedTicket_EmptyReturnsNil(t *testing.T) {
	lp := NewListPane(80, 20)
	if lp.SelectedTicket() != nil {
		t.Fatal("expected nil for empty list")
	}
}

func TestListPane_SelectionPreservedAfterRefresh(t *testing.T) {
	lp := NewListPane(80, 20)
	tickets := testTickets()
	lp.SetTickets(tickets)

	// Move cursor to second ticket
	lp.SetCursor(1)
	if lp.SelectedTicket().ID != "e5f6g7h8" {
		t.Fatalf("expected second ticket selected, got %q", lp.SelectedTicket().ID)
	}

	// Refresh with same tickets in different order
	refreshed := []model.Ticket{tickets[2], tickets[0], tickets[1]}
	lp.SetTickets(refreshed)

	// Should still have e5f6g7h8 selected
	selected := lp.SelectedTicket()
	if selected == nil || selected.ID != "e5f6g7h8" {
		id := ""
		if selected != nil {
			id = selected.ID
		}
		t.Fatalf("selection not preserved after refresh, got %q, want %q", id, "e5f6g7h8")
	}
}

func TestListPane_BuildRow_FormatsCorrectly(t *testing.T) {
	lp := NewListPane(80, 20)
	ticket := testTickets()[0] // TODO, a1b2c3d4, tom, 2d ago
	row := lp.BuildRow(ticket)

	if row[0] != "TODO" {
		t.Fatalf("row[0] (status) = %q, want %q", row[0], "TODO")
	}
	if row[1] != "a1b2c3d4" {
		t.Fatalf("row[1] (id) = %q, want %q", row[1], "a1b2c3d4")
	}
	if row[2] != "@tom" {
		t.Fatalf("row[2] (assignee) = %q, want %q", row[2], "@tom")
	}
	// row[3] is age — just check it's not empty
	if row[3] == "" {
		t.Fatal("row[3] (age) is empty")
	}
	if row[4] != "Fix login bug" {
		t.Fatalf("row[4] (title) = %q, want %q", row[4], "Fix login bug")
	}
}

func TestListPane_BuildRow_NoAssignee(t *testing.T) {
	lp := NewListPane(80, 20)
	ticket := testTickets()[2] // no assignee
	row := lp.BuildRow(ticket)

	if row[2] != "--" {
		t.Fatalf("row[2] (no assignee) = %q, want %q", row[2], "--")
	}
}

func TestListPane_Filtering(t *testing.T) {
	lp := NewListPane(80, 20)
	lp.SetTickets(testTickets())

	// Initially not filtering
	if lp.Filtering() {
		t.Fatal("should not be filtering initially")
	}
}
