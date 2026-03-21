package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"raptor/model"
	"strings"
	"testing"
)

func TestClient_CreateTicket(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		expected := "/api/workspaces/ws1/boards/bd1/tickets"
		if r.URL.Path != expected {
			t.Fatalf("expected %s, got %s", expected, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(model.Ticket{ID: "abc12345", Title: "Test"})
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "", "ws1", "bd1")
	ticket, err := c.CreateTicket("Test", "content", "")
	if err != nil {
		t.Fatal(err)
	}
	if ticket.ID != "abc12345" {
		t.Fatalf("expected ID abc12345, got %s", ticket.ID)
	}
}

func TestClient_ListTickets(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]model.Ticket{
			{ID: "a", Title: "One"},
			{ID: "b", Title: "Two"},
		})
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "", "ws1", "bd1")
	tickets, err := c.ListTickets(ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tickets) != 2 {
		t.Fatalf("expected 2 tickets, got %d", len(tickets))
	}
}

func TestClient_GetTicket(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(model.Ticket{ID: "abc", Title: "Found"})
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "", "ws1", "bd1")
	ticket, err := c.GetTicket("abc")
	if err != nil {
		t.Fatal(err)
	}
	if ticket.Title != "Found" {
		t.Fatalf("expected Found, got %s", ticket.Title)
	}
}

func TestClient_UpdateTicket(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(model.Ticket{ID: "abc", Title: "Updated"})
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "", "ws1", "bd1")
	ticket, err := c.UpdateTicket("abc", map[string]any{"title": "Updated"})
	if err != nil {
		t.Fatal(err)
	}
	if ticket.Title != "Updated" {
		t.Fatalf("expected Updated, got %s", ticket.Title)
	}
}

func TestClient_DeleteTicket(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "", "ws1", "bd1")
	err := c.DeleteTicket("abc")
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_AuthHeader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Fatalf("expected Bearer test-token, got %q", auth)
		}
		json.NewEncoder(w).Encode([]model.Ticket{})
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "test-token", "ws1", "bd1")
	_, err := c.ListTickets(ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_ScopedURLs(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode([]model.Ticket{})
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "", "ws1", "bd1")
	c.ListTickets(ListOptions{})
	expected := "/api/workspaces/ws1/boards/bd1/tickets"
	if gotPath != expected {
		t.Fatalf("expected %s, got %s", expected, gotPath)
	}
}

func TestClient_CreateWorkspace(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/workspaces/" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(model.Workspace{ID: "ws1", Name: "Team"})
	}))
	defer ts.Close()

	c := New(ts.URL, "")
	ws, err := c.CreateWorkspace("Team")
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != "Team" {
		t.Fatalf("expected Team, got %s", ws.Name)
	}
}

func TestClient_ListWorkspaces(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]model.Workspace{{ID: "ws1", Name: "Team"}})
	}))
	defer ts.Close()

	c := New(ts.URL, "")
	workspaces, err := c.ListWorkspaces()
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(workspaces))
	}
}

func TestClient_CreateBoard(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/workspaces/ws1/boards" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(model.Board{ID: "bd1", Name: "Sprint"})
	}))
	defer ts.Close()

	c := New(ts.URL, "")
	bd, err := c.CreateBoard("ws1", "Sprint", nil)
	if err != nil {
		t.Fatal(err)
	}
	if bd.Name != "Sprint" {
		t.Fatalf("expected Sprint, got %s", bd.Name)
	}
}

func TestClient_ListTickets_StatusEncoded(t *testing.T) {
	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		json.NewEncoder(w).Encode([]model.Ticket{})
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "", "ws1", "bd1")
	c.ListTickets(ListOptions{Status: "in_progress"})
	if !strings.Contains(gotQuery, "status=in_progress") {
		t.Fatalf("expected status=in_progress in query, got %q", gotQuery)
	}
}

func TestClient_SearchTickets(t *testing.T) {
	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		json.NewEncoder(w).Encode([]model.Ticket{{ID: "a", Title: "Found"}})
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "", "ws1", "bd1")
	tickets, err := c.SearchTickets("hello world")
	if err != nil {
		t.Fatal(err)
	}
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(tickets))
	}
	if !strings.Contains(gotQuery, "q=hello+world") {
		t.Fatalf("expected URL-encoded query, got %q", gotQuery)
	}
}

func TestClient_TicketStats(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"total":  3,
			"counts": map[string]any{"todo": 2, "done": 1},
		})
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "", "ws1", "bd1")
	result, err := c.TicketStats()
	if err != nil {
		t.Fatal(err)
	}
	if result["total"].(float64) != 3 {
		t.Fatalf("expected total 3, got %v", result["total"])
	}
}

func TestClient_TransportError(t *testing.T) {
	// Use a URL that will refuse connections to trigger transport errors
	c := NewScoped("http://127.0.0.1:1", "", "ws1", "bd1")

	if _, err := c.CreateTicket("x", "", ""); err == nil {
		t.Fatal("expected transport error from CreateTicket")
	}
	if _, err := c.ListTickets(ListOptions{}); err == nil {
		t.Fatal("expected transport error from ListTickets")
	}
	if _, err := c.SearchTickets("q"); err == nil {
		t.Fatal("expected transport error from SearchTickets")
	}
	if _, err := c.TicketStats(); err == nil {
		t.Fatal("expected transport error from TicketStats")
	}
	if _, err := c.GetTicket("x"); err == nil {
		t.Fatal("expected transport error from GetTicket")
	}
	if _, err := c.UpdateTicket("x", nil); err == nil {
		t.Fatal("expected transport error from UpdateTicket")
	}
	if err := c.DeleteTicket("x"); err == nil {
		t.Fatal("expected transport error from DeleteTicket")
	}

	c2 := New("http://127.0.0.1:1", "")
	if _, err := c2.CreateWorkspace("x"); err == nil {
		t.Fatal("expected transport error from CreateWorkspace")
	}
	if _, err := c2.ListWorkspaces(); err == nil {
		t.Fatal("expected transport error from ListWorkspaces")
	}
	if err := c2.DeleteWorkspace("x"); err == nil {
		t.Fatal("expected transport error from DeleteWorkspace")
	}
	if _, err := c2.ListWorkspaceMembers("x"); err == nil {
		t.Fatal("expected transport error from ListWorkspaceMembers")
	}
	if err := c2.InviteWorkspaceMember("x", "u"); err == nil {
		t.Fatal("expected transport error from InviteWorkspaceMember")
	}
	if err := c2.KickWorkspaceMember("x", "u"); err == nil {
		t.Fatal("expected transport error from KickWorkspaceMember")
	}
	if _, err := c2.CreateBoard("x", "b", nil); err == nil {
		t.Fatal("expected transport error from CreateBoard")
	}
	if _, err := c2.GetBoard("x", "b"); err == nil {
		t.Fatal("expected transport error from GetBoard")
	}
	if _, err := c2.ListBoards("x"); err == nil {
		t.Fatal("expected transport error from ListBoards")
	}
	if _, err := c2.UpdateBoard("x", "b", nil); err == nil {
		t.Fatal("expected transport error from UpdateBoard")
	}
	if err := c2.DeleteBoard("x", "b"); err == nil {
		t.Fatal("expected transport error from DeleteBoard")
	}
}

func TestClient_Decode_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "", "ws1", "bd1")
	_, err := c.ListTickets(ListOptions{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "decode response") {
		t.Fatalf("expected decode error, got %q", err.Error())
	}
}

func TestClient_Decode_UnexpectedStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "", "ws1", "bd1")
	_, err := c.ListTickets(ListOptions{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected status: 500") {
		t.Fatalf("expected unexpected status error, got %q", err.Error())
	}
}

func TestClient_ListTickets_MineFilter(t *testing.T) {
	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		json.NewEncoder(w).Encode([]model.Ticket{})
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "", "ws1", "bd1")
	c.ListTickets(ListOptions{Mine: true})
	if !strings.Contains(gotQuery, "mine=true") {
		t.Fatalf("expected mine=true in query, got %q", gotQuery)
	}
}

func TestClient_CreateTicket_WithAssignee(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["assignee"] != "alice" {
			t.Fatalf("expected assignee alice, got %q", body["assignee"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(model.Ticket{ID: "abc", Title: "Test", Assignee: "alice"})
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "", "ws1", "bd1")
	ticket, err := c.CreateTicket("Test", "content", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if ticket.Assignee != "alice" {
		t.Fatalf("expected assignee alice, got %s", ticket.Assignee)
	}
}

func TestClient_CreateBoard_WithStatuses(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		statuses, ok := body["statuses"].([]any)
		if !ok || len(statuses) != 3 {
			t.Fatalf("expected 3 statuses, got %v", body["statuses"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(model.Board{ID: "bd1", Name: "Sprint", Statuses: "backlog,active,done"})
	}))
	defer ts.Close()

	c := New(ts.URL, "")
	bd, err := c.CreateBoard("ws1", "Sprint", []string{"backlog", "active", "done"})
	if err != nil {
		t.Fatal(err)
	}
	if bd.Statuses != "backlog,active,done" {
		t.Fatalf("expected backlog,active,done, got %s", bd.Statuses)
	}
}

func TestClient_Decode_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "", "ws1", "bd1")
	_, err := c.ListTickets(ListOptions{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "run `raptor login`") {
		t.Fatalf("expected error to mention 'run `raptor login`', got %q", err.Error())
	}
}

func TestClient_GetTicket_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	c := NewScoped(ts.URL, "", "ws1", "bd1")
	_, err := c.GetTicket("missing123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected error to contain 'not found', got %q", err.Error())
	}
}

func TestClient_DeleteWorkspace(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/workspaces/ws1" {
			t.Fatalf("expected /api/workspaces/ws1, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := New(ts.URL, "")
	err := c.DeleteWorkspace("ws1")
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_ListWorkspaceMembers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workspaces/ws1/members" {
			t.Fatalf("expected /api/workspaces/ws1/members, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]model.WorkspaceMember{
			{WorkspaceID: "ws1", Username: "alice", Role: "owner"},
			{WorkspaceID: "ws1", Username: "bob", Role: "member"},
		})
	}))
	defer ts.Close()

	c := New(ts.URL, "")
	members, err := c.ListWorkspaceMembers("ws1")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if members[0].Username != "alice" {
		t.Fatalf("expected alice, got %s", members[0].Username)
	}
}

func TestClient_InviteWorkspaceMember(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/workspaces/ws1/members" {
			t.Fatalf("expected /api/workspaces/ws1/members, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	c := New(ts.URL, "")
	err := c.InviteWorkspaceMember("ws1", "bob")
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_InviteWorkspaceMember_Conflict(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer ts.Close()

	c := New(ts.URL, "")
	err := c.InviteWorkspaceMember("ws1", "bob")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "bob is already a member of this workspace") {
		t.Fatalf("expected conflict message, got %q", err.Error())
	}
}

func TestClient_KickWorkspaceMember(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/workspaces/ws1/members/bob" {
			t.Fatalf("expected /api/workspaces/ws1/members/bob, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := New(ts.URL, "")
	err := c.KickWorkspaceMember("ws1", "bob")
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_GetBoard(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workspaces/ws1/boards/bd1" {
			t.Fatalf("expected /api/workspaces/ws1/boards/bd1, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(model.Board{ID: "bd1", Name: "Sprint", Statuses: "todo,in_progress,done"})
	}))
	defer ts.Close()

	c := New(ts.URL, "")
	bd, err := c.GetBoard("ws1", "bd1")
	if err != nil {
		t.Fatal(err)
	}
	if bd.Name != "Sprint" {
		t.Fatalf("expected Sprint, got %s", bd.Name)
	}
	if bd.Statuses != "todo,in_progress,done" {
		t.Fatalf("expected todo,in_progress,done, got %s", bd.Statuses)
	}
}

func TestClient_ListBoards(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workspaces/ws1/boards" {
			t.Fatalf("expected /api/workspaces/ws1/boards, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]model.Board{
			{ID: "bd1", Name: "Sprint 1"},
			{ID: "bd2", Name: "Sprint 2"},
		})
	}))
	defer ts.Close()

	c := New(ts.URL, "")
	boards, err := c.ListBoards("ws1")
	if err != nil {
		t.Fatal(err)
	}
	if len(boards) != 2 {
		t.Fatalf("expected 2 boards, got %d", len(boards))
	}
}

func TestClient_DeleteBoard(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/workspaces/ws1/boards/bd1" {
			t.Fatalf("expected /api/workspaces/ws1/boards/bd1, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := New(ts.URL, "")
	err := c.DeleteBoard("ws1", "bd1")
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_UpdateBoard(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		expected := "/api/workspaces/ws1/boards/bd1"
		if r.URL.Path != expected {
			t.Fatalf("expected %s, got %s", expected, r.URL.Path)
		}
		json.NewEncoder(w).Encode(model.Board{ID: "bd1", Name: "Updated"})
	}))
	defer ts.Close()

	c := New(ts.URL, "")
	bd, err := c.UpdateBoard("ws1", "bd1", map[string]any{"name": "Updated"})
	if err != nil {
		t.Fatal(err)
	}
	if bd.Name != "Updated" {
		t.Fatalf("expected Updated, got %s", bd.Name)
	}
}
