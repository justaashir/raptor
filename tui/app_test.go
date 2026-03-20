package tui

import (
	"raptor/model"
	"testing"
	"time"
)

func sampleTickets() []model.Ticket {
	now := time.Now()
	return []model.Ticket{
		{ID: "aaaa1111", Title: "Task A", Status: model.Todo, Assignee: "tom", CreatedAt: now.Add(-2 * 24 * time.Hour), UpdatedAt: now},
		{ID: "bbbb2222", Title: "Task B", Status: model.InProgress, Assignee: "alice", CreatedAt: now.Add(-5 * time.Hour), UpdatedAt: now},
		{ID: "cccc3333", Title: "Task C", Status: model.Done, CreatedAt: now.Add(-7 * 24 * time.Hour), UpdatedAt: now},
		{ID: "dddd4444", Title: "Task D", Status: model.Todo, CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now},
	}
}

func TestApp_NewApp_CreatesListAndDetailPanes(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	if app.listPane == nil {
		t.Fatal("listPane should not be nil")
	}
	if app.detailPane == nil {
		t.Fatal("detailPane should not be nil")
	}
}

func TestApp_SetTickets_PopulatesListAndDetail(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	app.width = 120
	app.height = 40
	app.initPanes()
	app.SetTickets(sampleTickets())

	if app.listPane.RowCount() != 4 {
		t.Fatalf("expected 4 rows, got %d", app.listPane.RowCount())
	}

	selected := app.listPane.SelectedTicket()
	if selected == nil || selected.ID != "aaaa1111" {
		t.Fatal("first ticket should be selected after SetTickets")
	}
}

func TestApp_FocusedPane_DefaultIsList(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	if app.focused != focusList {
		t.Fatalf("expected focusList, got %d", app.focused)
	}
}

func TestApp_ToggleFocus_SwitchesBetweenPanes(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")

	app.toggleFocus()
	if app.focused != focusDetail {
		t.Fatalf("expected focusDetail after toggle, got %d", app.focused)
	}

	app.toggleFocus()
	if app.focused != focusList {
		t.Fatalf("expected focusList after second toggle, got %d", app.focused)
	}
}

func TestApp_SelectedTicket_ReturnsFromList(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	app.width = 120
	app.height = 40
	app.initPanes()
	app.SetTickets(sampleTickets())

	selected := app.SelectedTicket()
	if selected == nil || selected.ID != "aaaa1111" {
		t.Fatal("SelectedTicket should return first ticket")
	}
}

func TestApp_SelectedTicket_NilWhenEmpty(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	if app.SelectedTicket() != nil {
		t.Fatal("SelectedTicket should be nil when no tickets")
	}
}

func TestApp_AllTickets_StoredForStatusBar(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	app.width = 120
	app.height = 40
	app.initPanes()
	app.SetTickets(sampleTickets())

	if len(app.tickets) != 4 {
		t.Fatalf("expected 4 tickets stored, got %d", len(app.tickets))
	}
}

func TestApp_BoardAutoSelected_SetsStateAndReturnsCmd(t *testing.T) {
	app := NewApp("http://localhost:8080", "tok", "", "")
	app.width = 120
	app.height = 40
	app.initPanes()

	// Simulate receiving a boardAutoSelectedMsg (what fetchBoards should return
	// when there's a single workspace + single board)
	msg := boardAutoSelectedMsg{
		workspace:   "ws1",
		wsName:      "My Workspace",
		board:       "b1",
		boardName:   "My Board",
	}

	_, cmd := app.Update(msg)

	// State should be set
	if app.workspace != "ws1" {
		t.Fatalf("expected workspace 'ws1', got %q", app.workspace)
	}
	if app.board != "b1" {
		t.Fatalf("expected board 'b1', got %q", app.board)
	}
	if app.wsName != "My Workspace" {
		t.Fatalf("expected wsName 'My Workspace', got %q", app.wsName)
	}
	if app.boardName != "My Board" {
		t.Fatalf("expected boardName 'My Board', got %q", app.boardName)
	}
	if app.state != viewList {
		t.Fatalf("expected viewList state, got %d", app.state)
	}
	// Must return a command (tea.Batch of fetchTickets + listenWS)
	if cmd == nil {
		t.Fatal("expected non-nil cmd (should batch fetchTickets + listenWS)")
	}
}

func TestTicketCache_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/ticket_cache.json"

	tickets := sampleTickets()
	if err := SaveTicketCache(path, "b1", tickets); err != nil {
		t.Fatalf("SaveTicketCache: %v", err)
	}

	loaded, err := LoadTicketCache(path, "b1")
	if err != nil {
		t.Fatalf("LoadTicketCache: %v", err)
	}
	if len(loaded) != len(tickets) {
		t.Fatalf("expected %d tickets, got %d", len(tickets), len(loaded))
	}
	for i, tk := range loaded {
		if tk.ID != tickets[i].ID {
			t.Fatalf("ticket %d: expected ID %q, got %q", i, tickets[i].ID, tk.ID)
		}
		if tk.Title != tickets[i].Title {
			t.Fatalf("ticket %d: expected Title %q, got %q", i, tickets[i].Title, tk.Title)
		}
	}
}

func TestTicketCache_WrongBoard_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/ticket_cache.json"

	if err := SaveTicketCache(path, "b1", sampleTickets()); err != nil {
		t.Fatalf("SaveTicketCache: %v", err)
	}

	loaded, err := LoadTicketCache(path, "other-board")
	if err != nil {
		t.Fatalf("LoadTicketCache: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected 0 tickets for wrong board, got %d", len(loaded))
	}
}

func TestTicketCache_MissingFile_ReturnsEmpty(t *testing.T) {
	loaded, err := LoadTicketCache("/nonexistent/path", "b1")
	if err != nil {
		t.Fatalf("LoadTicketCache: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected 0 tickets for missing file, got %d", len(loaded))
	}
}

func TestApp_Init_LoadsCachedTickets(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/ticket_cache.json"

	tickets := sampleTickets()
	SaveTicketCache(path, "b1", tickets)

	app := NewApp("http://localhost:8080", "tok", "ws1", "b1")
	app.cachePath = path
	app.width = 120
	app.height = 40
	app.initPanes()

	// Init should load cached tickets synchronously before returning cmds
	app.Init()

	if len(app.tickets) != len(tickets) {
		t.Fatalf("expected %d cached tickets on Init, got %d", len(tickets), len(app.tickets))
	}
	if app.listPane.RowCount() != len(tickets) {
		t.Fatalf("expected %d rows in list, got %d", len(tickets), app.listPane.RowCount())
	}
}
