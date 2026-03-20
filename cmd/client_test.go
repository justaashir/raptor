package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"raptor/model"
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

	c := NewScopedClient(ts.URL, "", "ws1", "bd1")
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

	c := NewScopedClient(ts.URL, "", "ws1", "bd1")
	tickets, err := c.ListTickets("", false)
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

	c := NewScopedClient(ts.URL, "", "ws1", "bd1")
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

	c := NewScopedClient(ts.URL, "", "ws1", "bd1")
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

	c := NewScopedClient(ts.URL, "", "ws1", "bd1")
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

	c := NewScopedClient(ts.URL, "test-token", "ws1", "bd1")
	_, err := c.ListTickets("", false)
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

	c := NewScopedClient(ts.URL, "", "ws1", "bd1")
	c.ListTickets("", false)
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

	c := NewClient(ts.URL, "")
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

	c := NewClient(ts.URL, "")
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

	c := NewClient(ts.URL, "")
	bd, err := c.CreateBoard("ws1", "Sprint")
	if err != nil {
		t.Fatal(err)
	}
	if bd.Name != "Sprint" {
		t.Fatalf("expected Sprint, got %s", bd.Name)
	}
}
