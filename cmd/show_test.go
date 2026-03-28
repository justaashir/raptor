package cmd

import (
	"raptor/model"
	"regexp"
	"strings"
	"testing"
	"time"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

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
	plain := stripANSI(got)
	for _, want := range []string{"abc12345", "Fix the login bug", "todo", "alice", "bob"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("expected %q in output, got:\n%s", want, plain)
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
	plain := stripANSI(got)
	if !strings.Contains(plain, "body") {
		t.Fatalf("expected content in output, got:\n%s", plain)
	}
}

func TestRenderTicketView_ContainsANSI(t *testing.T) {
	tk := model.Ticket{
		ID:      "abc12345",
		Title:   "Test",
		Status:  model.Todo,
		Content: "Some **bold** text.",
	}
	got, err := renderTicketView(tk)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "\033[") {
		t.Fatalf("expected ANSI escape codes in output, got:\n%s", got)
	}
}

func TestRenderTicketView_EmptyContent_NoSeparator(t *testing.T) {
	tk := model.Ticket{
		ID:     "abc12345",
		Title:  "No body",
		Status: model.Todo,
	}
	got, err := renderTicketView(tk)
	if err != nil {
		t.Fatal(err)
	}
	plain := stripANSI(got)
	if strings.Contains(plain, "---") {
		t.Fatalf("should not contain separator when content is empty, got:\n%s", plain)
	}
}
