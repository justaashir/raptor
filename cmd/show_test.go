package cmd

import (
	"raptor/model"
	"strings"
	"testing"
	"time"
)

func TestRenderTicketView_ShowsMetadata(t *testing.T) {
	tk := model.Ticket{
		ID:        "abc12345",
		Title:     "Fix the login bug",
		Status:    model.Todo,
		CreatedBy: "alice",
		Assignee:  "bob",
		CreatedAt: time.Date(2026, 3, 28, 10, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 28, 11, 0, 0, 0, time.UTC),
	}
	got, err := renderTicketView(tk)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"abc12345", "Fix the login bug", "todo", "alice", "bob"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestRenderTicketView_IncludesContent(t *testing.T) {
	tk := model.Ticket{
		ID:      "abc12345",
		Title:   "Test",
		Status:  model.Todo,
		Content: "This is the **body** of the ticket.",
	}
	got, err := renderTicketView(tk)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "body") {
		t.Fatalf("expected content in output, got:\n%s", got)
	}
}
