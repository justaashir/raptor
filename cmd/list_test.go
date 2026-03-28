package cmd

import (
	"raptor/model"
	"testing"
)

func TestFormatTicketTable_EmptySlice_ReturnsHeaderOnly(t *testing.T) {
	got := formatTicketTable(nil)
	want := "ID\tSTATUS\tASSIGNEE\tTITLE\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFormatTicketTable_OneTicket_ReturnsHeaderAndRow(t *testing.T) {
	tickets := []model.Ticket{
		{ID: "abc12345", Status: model.Todo, Assignee: "alice", Title: "Fix bug"},
	}
	got := formatTicketTable(tickets)
	want := "ID\tSTATUS\tASSIGNEE\tTITLE\nabc12345\ttodo\talice\tFix bug\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
