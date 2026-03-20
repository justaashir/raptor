package server

import (
	"raptor/model"
	"testing"
)

func TestDB_CreateAndGetTicket(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	ticket := model.NewTicket("Test ticket", "some content", "alice")
	err = db.CreateTicket(ticket)
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	got, err := db.GetTicket(ticket.ID)
	if err != nil {
		t.Fatalf("failed to get ticket: %v", err)
	}
	if got.ID != ticket.ID {
		t.Fatalf("expected ID %q, got %q", ticket.ID, got.ID)
	}
	if got.Title != "Test ticket" {
		t.Fatalf("expected title %q, got %q", "Test ticket", got.Title)
	}
	if got.Content != "some content" {
		t.Fatalf("expected content %q, got %q", "some content", got.Content)
	}
	if got.Status != model.Todo {
		t.Fatalf("expected status %q, got %q", model.Todo, got.Status)
	}
	if got.CreatedBy != "alice" {
		t.Fatalf("expected created_by %q, got %q", "alice", got.CreatedBy)
	}
}

func newTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestDB_ListTickets_Empty(t *testing.T) {
	db := newTestDB(t)
	tickets, err := db.ListTickets("")
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(tickets) != 0 {
		t.Fatalf("expected 0 tickets, got %d", len(tickets))
	}
}

func TestDB_ListTickets_FilterByStatus(t *testing.T) {
	db := newTestDB(t)
	t1 := model.NewTicket("Todo task", "", "")
	t2 := model.NewTicket("Done task", "", "")
	t2.Status = model.Done
	db.CreateTicket(t1)
	db.CreateTicket(t2)

	tickets, err := db.ListTickets("todo")
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(tickets))
	}
	if tickets[0].Title != "Todo task" {
		t.Fatalf("expected %q, got %q", "Todo task", tickets[0].Title)
	}
}

func TestDB_UpdateTicket(t *testing.T) {
	db := newTestDB(t)
	ticket := model.NewTicket("Original", "", "")
	db.CreateTicket(ticket)

	err := db.UpdateTicket(ticket.ID, map[string]any{
		"title":  "Updated",
		"status": "in_progress",
	})
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	got, _ := db.GetTicket(ticket.ID)
	if got.Title != "Updated" {
		t.Fatalf("expected title %q, got %q", "Updated", got.Title)
	}
	if got.Status != model.InProgress {
		t.Fatalf("expected status %q, got %q", model.InProgress, got.Status)
	}
}

func TestDB_DeleteTicket(t *testing.T) {
	db := newTestDB(t)
	ticket := model.NewTicket("To delete", "", "")
	db.CreateTicket(ticket)

	err := db.DeleteTicket(ticket.ID)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err = db.GetTicket(ticket.ID)
	if err == nil {
		t.Fatal("expected error getting deleted ticket")
	}
}

func TestDB_CreateWorkspace(t *testing.T) {
	db := newTestDB(t)
	err := db.CreateWorkspace("ws123456", "My Team", "alice")
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	workspaces, err := db.ListWorkspacesForUser("alice")
	if err != nil {
		t.Fatalf("failed to list workspaces: %v", err)
	}
	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(workspaces))
	}
	if workspaces[0].Name != "My Team" {
		t.Fatalf("expected name %q, got %q", "My Team", workspaces[0].Name)
	}
}

func TestDB_AssigneeField(t *testing.T) {
	db := newTestDB(t)
	ticket := model.NewTicket("Assigned task", "", "alice")
	ticket.Assignee = "bob"
	db.CreateTicket(ticket)

	got, err := db.GetTicket(ticket.ID)
	if err != nil {
		t.Fatalf("failed to get ticket: %v", err)
	}
	if got.Assignee != "bob" {
		t.Fatalf("expected assignee %q, got %q", "bob", got.Assignee)
	}
	if got.CreatedBy != "alice" {
		t.Fatalf("expected created_by %q, got %q", "alice", got.CreatedBy)
	}
}
