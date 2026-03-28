package cmd

import (
	"raptor/model"
	"strings"
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

func TestRenderTicketTable_AlignedColumns(t *testing.T) {
	tickets := []model.Ticket{
		{ID: "aaa11111", Status: model.Todo, Assignee: "alice", Title: "First"},
		{ID: "bbb22222", Status: model.InProgress, Assignee: "bob", Title: "Second"},
	}
	got := renderTicketTable(tickets)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines (header + 2 rows), got %d: %q", len(lines), got)
	}
	// Header should contain column names
	if !strings.Contains(lines[0], "ID") || !strings.Contains(lines[0], "TITLE") {
		t.Fatalf("header missing column names: %q", lines[0])
	}
	// All STATUS columns should start at the same position
	headerPos := strings.Index(lines[0], "STATUS")
	row1Pos := strings.Index(lines[1], "todo")
	row2Pos := strings.Index(lines[2], "in_progress")
	if headerPos != row1Pos || headerPos != row2Pos {
		t.Fatalf("STATUS column not aligned: header=%d, row1=%d, row2=%d", headerPos, row1Pos, row2Pos)
	}
}

func TestRenderTicketTable_ShowsCountSummary(t *testing.T) {
	tickets := []model.Ticket{
		{ID: "a", Status: model.Todo, Title: "One"},
		{ID: "b", Status: model.Done, Title: "Two"},
		{ID: "c", Status: model.Todo, Title: "Three"},
	}
	got := renderTicketTable(tickets)
	if !strings.Contains(got, "3 tickets") {
		t.Fatalf("expected count summary '3 tickets', got %q", got)
	}
}

func TestListCmd_HasLsAlias(t *testing.T) {
	aliases := listCmd.Aliases
	found := false
	for _, a := range aliases {
		if a == "ls" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'ls' alias, got %v", aliases)
	}
}

func TestRenderTicketTable_SingleTicket_SingularCount(t *testing.T) {
	tickets := []model.Ticket{
		{ID: "a", Status: model.Todo, Title: "One"},
	}
	got := renderTicketTable(tickets)
	if !strings.Contains(got, "1 ticket") {
		t.Fatalf("expected '1 ticket', got %q", got)
	}
	if strings.Contains(got, "1 tickets") {
		t.Fatalf("should be singular '1 ticket', not '1 tickets'")
	}
}
