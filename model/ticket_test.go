package model

import (
	"testing"
	"time"
)

func TestNewTicket_HasIDTitleAndStatus(t *testing.T) {
	ticket := NewTicket("My first task", "", "alice")

	if ticket.ID == "" {
		t.Fatal("expected ticket to have an ID")
	}
	if len(ticket.ID) != 12 {
		t.Fatalf("expected 12-char ID, got %d: %q", len(ticket.ID), ticket.ID)
	}
	if ticket.Title != "My first task" {
		t.Fatalf("expected title %q, got %q", "My first task", ticket.Title)
	}
	if ticket.Status != Todo {
		t.Fatalf("expected status %q, got %q", Todo, ticket.Status)
	}
	if ticket.CreatedBy != "alice" {
		t.Fatalf("expected created_by %q, got %q", "alice", ticket.CreatedBy)
	}
	if ticket.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
	if time.Since(ticket.CreatedAt) > time.Second {
		t.Fatal("expected CreatedAt to be recent")
	}
}

func TestBoard_ValidStatus_AcceptsConfiguredStatuses(t *testing.T) {
	b := Board{Statuses: "backlog,dev,done"}
	for _, s := range []string{"backlog", "dev", "done"} {
		if !b.ValidStatus(s) {
			t.Fatalf("expected %q to be valid for board", s)
		}
	}
}

func TestBoard_ValidStatus_RejectsInvalidStatus(t *testing.T) {
	b := Board{Statuses: "backlog,dev,done"}
	if b.ValidStatus("banana") {
		t.Fatal("expected 'banana' to be invalid")
	}
}
