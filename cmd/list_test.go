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

func TestFormatTicketTable_MultipleTickets_AllRowsPresent(t *testing.T) {
	tickets := []model.Ticket{
		{ID: "aaa11111", Status: model.Todo, Assignee: "alice", Title: "First"},
		{ID: "bbb22222", Status: model.InProgress, Assignee: "bob", Title: "Second"},
		{ID: "ccc33333", Status: model.Done, Assignee: "", Title: "Third"},
	}
	got := formatTicketTable(tickets)
	want := "ID\tSTATUS\tASSIGNEE\tTITLE\n" +
		"aaa11111\ttodo\talice\tFirst\n" +
		"bbb22222\tin_progress\tbob\tSecond\n" +
		"ccc33333\tdone\t\tThird\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
