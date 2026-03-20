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
