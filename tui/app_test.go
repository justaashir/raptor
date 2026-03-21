package tui

import (
	"raptor/model"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

func TestApp_WorkspacesMsg_TransitionsToWorkspaceSelect(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	msg := workspacesMsg{
		workspaces: []model.Workspace{
			{ID: "ws1", Name: "Team Alpha"},
			{ID: "ws2", Name: "Team Beta"},
		},
	}
	app.Update(msg)
	if app.state != viewWorkspaceSelect {
		t.Fatalf("expected viewWorkspaceSelect, got %d", app.state)
	}
	if len(app.wsChoices) != 2 {
		t.Fatalf("expected 2 workspace choices, got %d", len(app.wsChoices))
	}
}

func TestApp_WorkspaceSelector_ShowsWorkspaceNames(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	app.state = viewWorkspaceSelect
	app.wsChoices = []model.Workspace{
		{ID: "ws1", Name: "Team Alpha"},
		{ID: "ws2", Name: "Team Beta"},
	}
	app.wsCursor = 0
	view := app.View()
	if !strings.Contains(view, "Select a workspace") {
		t.Fatal("should show 'Select a workspace' title")
	}
	if !strings.Contains(view, "Team Alpha") {
		t.Fatal("should show workspace name 'Team Alpha'")
	}
	if !strings.Contains(view, "Team Beta") {
		t.Fatal("should show workspace name 'Team Beta'")
	}
}

func TestApp_WorkspaceSelector_NavigatesUpDown(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	app.state = viewWorkspaceSelect
	app.wsChoices = []model.Workspace{
		{ID: "ws1", Name: "Team Alpha"},
		{ID: "ws2", Name: "Team Beta"},
		{ID: "ws3", Name: "Team Gamma"},
	}
	app.wsCursor = 0

	// Move down
	app.Update(tea.KeyMsg{Type: tea.KeyDown})
	if app.wsCursor != 1 {
		t.Fatalf("expected cursor 1 after down, got %d", app.wsCursor)
	}

	// Move down again
	app.Update(tea.KeyMsg{Type: tea.KeyDown})
	if app.wsCursor != 2 {
		t.Fatalf("expected cursor 2 after second down, got %d", app.wsCursor)
	}

	// Should not go past last
	app.Update(tea.KeyMsg{Type: tea.KeyDown})
	if app.wsCursor != 2 {
		t.Fatalf("expected cursor 2 (clamped), got %d", app.wsCursor)
	}

	// Move up
	app.Update(tea.KeyMsg{Type: tea.KeyUp})
	if app.wsCursor != 1 {
		t.Fatalf("expected cursor 1 after up, got %d", app.wsCursor)
	}
}

func TestApp_WorkspaceSelector_EnterSetsWorkspaceAndFetchesBoards(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	app.state = viewWorkspaceSelect
	app.wsChoices = []model.Workspace{
		{ID: "ws1", Name: "Team Alpha"},
		{ID: "ws2", Name: "Team Beta"},
	}
	app.wsCursor = 1 // select Team Beta

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if app.workspace != "ws2" {
		t.Fatalf("expected workspace 'ws2', got '%s'", app.workspace)
	}
	if app.wsName != "Team Beta" {
		t.Fatalf("expected wsName 'Team Beta', got '%s'", app.wsName)
	}
	if cmd == nil {
		t.Fatal("expected a command to fetch boards")
	}
}

func TestApp_PressW_FromListView_FetchesWorkspaces(t *testing.T) {
	app := NewApp("http://localhost:8080", "tok", "ws1", "b1")
	app.state = viewList
	app.width = 120
	app.height = 40
	app.initPanes()

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if cmd == nil {
		t.Fatal("expected a command to fetch workspaces when pressing 'w'")
	}
}

func TestApp_SingleWorkspace_AutoSelectsAndShowsBoardSelector(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "", "")
	// Simulate receiving a workspacesMsg with only 1 workspace
	msg := workspacesMsg{
		workspaces: []model.Workspace{
			{ID: "ws1", Name: "Only Workspace"},
		},
	}
	app.Update(msg)
	// Should auto-select the workspace and NOT show workspace selector
	if app.workspace != "ws1" {
		t.Fatalf("expected workspace auto-selected to 'ws1', got '%s'", app.workspace)
	}
	if app.wsName != "Only Workspace" {
		t.Fatalf("expected wsName 'Only Workspace', got '%s'", app.wsName)
	}
	// Should NOT be in workspace select state
	if app.state == viewWorkspaceSelect {
		t.Fatal("should not show workspace selector when only 1 workspace")
	}
}

func TestApp_SingleBoard_AutoSelectsAndGoesToList(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "ws1", "")
	app.wsName = "Team Alpha"
	msg := boardsMsg{
		boards:    []model.Board{{ID: "b1", Name: "Sprint 1", WorkspaceID: "ws1"}},
		workspace: "Team Alpha",
	}
	_, cmd := app.Update(msg)
	if app.board != "b1" {
		t.Fatalf("expected board auto-selected to 'b1', got '%s'", app.board)
	}
	if app.boardName != "Sprint 1" {
		t.Fatalf("expected boardName 'Sprint 1', got '%s'", app.boardName)
	}
	if app.state != viewList {
		t.Fatal("should transition to viewList when only 1 board")
	}
	if cmd == nil {
		t.Fatal("expected a command to fetch tickets")
	}
}

func TestApp_BoardSelector_EscGoesBackToWorkspaceSelector(t *testing.T) {
	app := NewApp("http://localhost:8080", "", "ws1", "")
	app.state = viewBoardSelect
	app.boardChoices = []model.Board{
		{ID: "b1", Name: "Board 1"},
		{ID: "b2", Name: "Board 2"},
	}

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected a command to fetch workspaces")
	}
	// workspace should be cleared so the full flow restarts
	if app.workspace != "" {
		t.Fatalf("expected workspace cleared, got '%s'", app.workspace)
	}
}

func TestApp_PressN_FromListView_ReturnsCreateCmd(t *testing.T) {
	app := NewApp("http://localhost:8080", "tok", "ws1", "b1")
	app.state = viewList
	app.width = 120
	app.height = 40
	app.initPanes()

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Fatal("expected a command when pressing 'n'")
	}
}

func TestApp_TicketCreatedMsg_RefreshesTickets(t *testing.T) {
	app := NewApp("http://localhost:8080", "tok", "ws1", "b1")
	app.state = viewList

	_, cmd := app.Update(ticketCreatedMsg{})
	if cmd == nil {
		t.Fatal("expected a refresh command after ticket creation")
	}
}

func TestApp_CreateView_RendersAsFloatingOverlay(t *testing.T) {
	app := NewApp("http://localhost:8080", "tok", "ws1", "b1")
	app.width = 120
	app.height = 40
	app.initPanes()
	app.SetTickets(sampleTickets())

	// Trigger create form
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if app.state != viewCreate {
		t.Fatalf("expected viewCreate state, got %d", app.state)
	}

	view := app.View()

	// Should contain the "New ticket" title
	if !strings.Contains(view, "New ticket") {
		t.Fatal("create view should contain 'New ticket' title")
	}

	// Should contain rounded border (floating window)
	if !strings.Contains(view, "╭") {
		t.Fatal("create view should have floating window border")
	}

	// Should contain the column header from the background (proves background is rendered)
	if !strings.Contains(view, "STATUS") {
		t.Fatal("create view should show background column header")
	}
}

func TestApp_CreateView_CancelReturnsToListWithoutOverlay(t *testing.T) {
	app := NewApp("http://localhost:8080", "tok", "ws1", "b1")
	app.width = 120
	app.height = 40
	app.boardName = "Sprint"
	app.initPanes()
	app.SetTickets(sampleTickets())

	// Enter create mode
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if app.state != viewCreate {
		t.Fatalf("expected viewCreate, got %d", app.state)
	}

	// Cancel with esc
	app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if app.state != viewList {
		t.Fatalf("expected viewList after cancel, got %d", app.state)
	}

	view := app.View()
	if !strings.Contains(view, "Sprint") {
		t.Fatal("after cancel, should show normal board view")
	}
	if strings.Contains(view, "New ticket") {
		t.Fatal("after cancel, should not show create overlay")
	}
}

func TestApp_CreateView_HandlesWindowResize(t *testing.T) {
	app := NewApp("http://localhost:8080", "tok", "ws1", "b1")
	app.width = 120
	app.height = 40
	app.initPanes()
	app.SetTickets(sampleTickets())

	// Enter create mode
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Resize while in create mode
	app.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	if app.width != 200 || app.height != 50 {
		t.Fatalf("expected dimensions 200x50, got %dx%d", app.width, app.height)
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

func TestApp_SingleBoardAutoSelected_SetsStateAndReturnsCmd(t *testing.T) {
	app := NewApp("http://localhost:8080", "tok", "", "")
	app.width = 120
	app.height = 40
	app.initPanes()

	// Simulate receiving a boardsMsg with a single board (auto-selects)
	msg := boardsMsg{
		boards: []model.Board{
			{ID: "b1", Name: "My Board", WorkspaceID: "ws1"},
		},
		workspace: "My Workspace",
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

func TestWSURL(t *testing.T) {
	tests := []struct {
		serverURL string
		token     string
		expected  string
	}{
		{"http://localhost:8080", "tok123", "ws://localhost:8080/ws?token=tok123"},
		{"https://raptor.example.com", "abc", "wss://raptor.example.com/ws?token=abc"},
		{"http://httpbin.org", "xyz", "ws://httpbin.org/ws?token=xyz"},
	}
	for _, tt := range tests {
		got := wsURL(tt.serverURL, tt.token)
		if got != tt.expected {
			t.Errorf("wsURL(%q, %q) = %q, want %q", tt.serverURL, tt.token, got, tt.expected)
		}
	}
}
