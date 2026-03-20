package model

import (
	"testing"
	"time"
)

func TestNewTicket_HasIDTitleAndStatus(t *testing.T) {
	ticket := NewTicket("My first task", "")

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
	if ticket.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
	if time.Since(ticket.CreatedAt) > time.Second {
		t.Fatal("expected CreatedAt to be recent")
	}
}
