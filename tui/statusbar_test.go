package tui

import (
	"raptor/model"
	"strings"
	"testing"
	"time"
)

func TestRenderStatusBar_ShowsTicketCounts(t *testing.T) {
	tickets := []model.Ticket{
		{ID: "1", Status: model.Todo},
		{ID: "2", Status: model.Todo},
		{ID: "3", Status: model.InProgress},
		{ID: "4", Status: model.Done},
	}

	bar := RenderStatusBar(tickets, "my-board", 80)

	if !strings.Contains(bar, "2") { // 2 todo
		t.Fatal("should contain todo count")
	}
	if !strings.Contains(bar, "1") { // 1 in_progress
		t.Fatal("should contain in_progress count")
	}
	if !strings.Contains(bar, "my-board") {
		t.Fatal("should contain board name")
	}
	if !strings.Contains(bar, "4 tickets") {
		t.Fatalf("should contain total ticket count, got %q", bar)
	}
}

func TestRenderStatusBar_ShowsKeybindHints(t *testing.T) {
	bar := RenderStatusBar(nil, "board", 80)

	if !strings.Contains(bar, "tab") {
		t.Fatal("should contain tab hint")
	}
	if !strings.Contains(bar, "quit") || !strings.Contains(bar, "q") {
		t.Fatal("should contain quit hint")
	}
}

func TestRenderStatusBar_EmptyTickets(t *testing.T) {
	bar := RenderStatusBar(nil, "board", 80)

	if !strings.Contains(bar, "0 tickets") {
		t.Fatalf("should show 0 tickets, got %q", bar)
	}
}

func TestCountByStatus(t *testing.T) {
	now := time.Now()
	tickets := []model.Ticket{
		{ID: "1", Status: model.Todo, CreatedAt: now},
		{ID: "2", Status: model.Todo, CreatedAt: now},
		{ID: "3", Status: model.InProgress, CreatedAt: now},
		{ID: "4", Status: model.Done, CreatedAt: now},
	}

	counts := CountByStatus(tickets)

	if counts[model.Todo] != 2 {
		t.Fatalf("Todo count = %d, want 2", counts[model.Todo])
	}
	if counts[model.InProgress] != 1 {
		t.Fatalf("InProgress count = %d, want 1", counts[model.InProgress])
	}
	if counts[model.Done] != 1 {
		t.Fatalf("Done count = %d, want 1", counts[model.Done])
	}
}

func TestRenderStatusBar_DeterministicOrder(t *testing.T) {
	tickets := []model.Ticket{
		{Status: "zzz"}, {Status: "zzz"},
		{Status: "aaa"},
	}
	first := RenderStatusBar(tickets, "Board", 80)
	for i := 0; i < 10; i++ {
		got := RenderStatusBar(tickets, "Board", 80)
		if got != first {
			t.Fatalf("render %d differs from first render — non-deterministic ordering", i)
		}
	}
}
