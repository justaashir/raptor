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
	if len(ticket.ID) != 36 {
		t.Fatalf("expected 36-char UUID, got %d: %q", len(ticket.ID), ticket.ID)
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

func TestNewTicket_UniqueIDs(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		tk := NewTicket("title", "", "user")
		if seen[tk.ID] {
			t.Fatalf("duplicate ID: %s", tk.ID)
		}
		if len(tk.ID) < 20 {
			t.Fatalf("ID too short (%d chars): %s — expected full UUID", len(tk.ID), tk.ID)
		}
		seen[tk.ID] = true
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
