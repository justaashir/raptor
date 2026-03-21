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
