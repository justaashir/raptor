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
	if len(ticket.ID) != 8 {
		t.Fatalf("expected 8-char ID, got %d: %q", len(ticket.ID), ticket.ID)
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

func TestValidStatus_AcceptsTodoInProgressDone(t *testing.T) {
	for _, s := range []Status{Todo, InProgress, Done} {
		if !ValidStatus(s) {
			t.Fatalf("expected %q to be valid", s)
		}
	}
}

func TestValidStatus_RejectsGarbage(t *testing.T) {
	if ValidStatus("banana") {
		t.Fatal("expected 'banana' to be invalid")
	}
}

func TestValidStatus_RejectsClosed(t *testing.T) {
	if ValidStatus("closed") {
		t.Fatal("expected 'closed' to be invalid — closed status removed")
	}
}
