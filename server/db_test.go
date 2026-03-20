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

func TestDB_WorkspaceMembers(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "Team", "alice")

	// alice is already owner from CreateWorkspace
	role, err := db.GetMemberRole("ws1", "alice")
	if err != nil {
		t.Fatalf("failed to get role: %v", err)
	}
	if role != "owner" {
		t.Fatalf("expected owner, got %q", role)
	}

	// Add bob as member
	err = db.AddWorkspaceMember("ws1", "bob", "member")
	if err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	members, err := db.ListWorkspaceMembers("ws1")
	if err != nil {
		t.Fatalf("failed to list members: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}

	// bob can see the workspace
	workspaces, _ := db.ListWorkspacesForUser("bob")
	if len(workspaces) != 1 {
		t.Fatalf("expected bob to see 1 workspace, got %d", len(workspaces))
	}
}

func TestDB_UpdateMemberRole(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "Team", "alice")
	db.AddWorkspaceMember("ws1", "bob", "member")

	err := db.UpdateMemberRole("ws1", "bob", "admin")
	if err != nil {
		t.Fatalf("failed to update role: %v", err)
	}
	role, _ := db.GetMemberRole("ws1", "bob")
	if role != "admin" {
		t.Fatalf("expected admin, got %q", role)
	}
}

func TestDB_RemoveWorkspaceMember(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "Team", "alice")
	db.AddWorkspaceMember("ws1", "bob", "member")

	err := db.RemoveWorkspaceMember("ws1", "bob")
	if err != nil {
		t.Fatalf("failed to remove member: %v", err)
	}
	members, _ := db.ListWorkspaceMembers("ws1")
	if len(members) != 1 {
		t.Fatalf("expected 1 member after remove, got %d", len(members))
	}
}

func TestDB_DeleteWorkspace(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "Team", "alice")
	err := db.DeleteWorkspace("ws1")
	if err != nil {
		t.Fatalf("failed to delete workspace: %v", err)
	}
	workspaces, _ := db.ListWorkspacesForUser("alice")
	if len(workspaces) != 0 {
		t.Fatalf("expected 0 workspaces after delete, got %d", len(workspaces))
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
