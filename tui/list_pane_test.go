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

func TestFormatStatus_CustomStatus(t *testing.T) {
	got := FormatStatus(model.Status("review"))
	if got != "review" {
		t.Fatalf("FormatStatus(review) = %q, want %q", got, "review")
	}
}

func TestSortTickets_DoneAtBottom(t *testing.T) {
	now := time.Now()
	tickets := []model.Ticket{
		{ID: "done1", Title: "Done first", Status: model.Done, CreatedAt: now},
		{ID: "todo1", Title: "Todo", Status: model.Todo, CreatedAt: now},
		{ID: "done2", Title: "Done second", Status: model.Done, CreatedAt: now},
		{ID: "prog1", Title: "In progress", Status: model.InProgress, CreatedAt: now},
	}
	sorted := sortTicketsDoneLast(tickets)
	// Non-done tickets should come first
	if sorted[0].ID != "todo1" {
		t.Fatalf("sorted[0] = %q, want todo1", sorted[0].ID)
	}
	if sorted[1].ID != "prog1" {
		t.Fatalf("sorted[1] = %q, want prog1", sorted[1].ID)
	}
	// Done tickets should come last
	if sorted[2].ID != "done1" {
		t.Fatalf("sorted[2] = %q, want done1", sorted[2].ID)
	}
	if sorted[3].ID != "done2" {
		t.Fatalf("sorted[3] = %q, want done2", sorted[3].ID)
	}
}

func TestListPane_SetTickets_DoneTicketsAppearLast(t *testing.T) {
	now := time.Now()
	tickets := []model.Ticket{
		{ID: "done1", Title: "Done first", Status: model.Done, CreatedAt: now},
		{ID: "todo1", Title: "Todo task", Status: model.Todo, CreatedAt: now},
		{ID: "prog1", Title: "In progress", Status: model.InProgress, CreatedAt: now},
	}
	lp := NewListPane(80, 20)
	lp.SetTickets(tickets)

	// First selected ticket should be the todo (non-done come first)
	selected := lp.SelectedTicket()
	if selected == nil || selected.ID != "todo1" {
		id := ""
		if selected != nil {
			id = selected.ID
		}
		t.Fatalf("first ticket should be todo1, got %q", id)
	}

	// Move to last item — should be the done ticket
	lp.SetCursor(2)
	selected = lp.SelectedTicket()
	if selected == nil || selected.ID != "done1" {
		id := ""
		if selected != nil {
			id = selected.ID
		}
		t.Fatalf("last ticket should be done1, got %q", id)
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
