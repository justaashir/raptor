package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"raptor/model"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	hub := NewHub()
	go hub.Run()
	return NewServer(db, hub)
}

func TestServer_CreateAndListTickets(t *testing.T) {
	srv := newTestServer(t)

	// Create a ticket
	body := `{"title":"Test ticket","content":"# Hello"}`
	req := httptest.NewRequest("POST", "/api/tickets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created model.Ticket
	json.NewDecoder(w.Body).Decode(&created)
	if created.ID == "" {
		t.Fatal("expected ticket to have an ID")
	}

	// List tickets
	req = httptest.NewRequest("GET", "/api/tickets", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var tickets []model.Ticket
	json.NewDecoder(w.Body).Decode(&tickets)
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(tickets))
	}
}

func TestServer_GetTicket(t *testing.T) {
	srv := newTestServer(t)

	ticket := model.NewTicket("Get me", "content", "")
	srv.db.CreateTicket(ticket)

	req := httptest.NewRequest("GET", "/api/tickets/"+ticket.ID, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var got model.Ticket
	json.NewDecoder(w.Body).Decode(&got)
	if got.Title != "Get me" {
		t.Fatalf("expected %q, got %q", "Get me", got.Title)
	}
}

func TestServer_UpdateTicket(t *testing.T) {
	srv := newTestServer(t)

	ticket := model.NewTicket("Original", "", "")
	srv.db.CreateTicket(ticket)

	body := `{"title":"Updated","status":"done"}`
	req := httptest.NewRequest("PATCH", "/api/tickets/"+ticket.ID, strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	got, _ := srv.db.GetTicket(ticket.ID)
	if got.Title != "Updated" {
		t.Fatalf("expected %q, got %q", "Updated", got.Title)
	}
}

func TestServer_DeleteTicket(t *testing.T) {
	srv := newTestServer(t)

	ticket := model.NewTicket("To delete", "", "")
	srv.db.CreateTicket(ticket)

	req := httptest.NewRequest("DELETE", "/api/tickets/"+ticket.ID, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	_, err := srv.db.GetTicket(ticket.ID)
	if err == nil {
		t.Fatal("expected ticket to be deleted")
	}
}

func TestServer_GetTicket_NotFound(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/tickets/nonexist", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestServer_CreateWorkspace(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := GenerateToken("alice", "secret")

	body := `{"name":"My Team"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var ws struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	json.NewDecoder(w.Body).Decode(&ws)
	if ws.Name != "My Team" {
		t.Fatalf("expected name My Team, got %q", ws.Name)
	}
	if ws.ID == "" {
		t.Fatal("expected workspace to have an ID")
	}
}

func TestServer_ListWorkspaces(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := GenerateToken("alice", "secret")

	// Create a workspace
	body := `{"name":"Team A"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// List workspaces
	req = httptest.NewRequest("GET", "/api/workspaces/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var workspaces []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	json.NewDecoder(w.Body).Decode(&workspaces)
	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(workspaces))
	}
}

func TestServer_CreatedByFromAuth(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := GenerateToken("alice", "secret")

	body := `{"title":"Auth ticket","content":""}`
	req := httptest.NewRequest("POST", "/api/tickets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created model.Ticket
	json.NewDecoder(w.Body).Decode(&created)
	if created.CreatedBy != "alice" {
		t.Fatalf("expected created_by=alice, got %q", created.CreatedBy)
	}
}

func TestServer_ListMine(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice", "bob"})
	tokenAlice := GenerateToken("alice", "secret")
	tokenBob := GenerateToken("bob", "secret")

	// Alice creates a ticket
	body := `{"title":"Alice ticket"}`
	req := httptest.NewRequest("POST", "/api/tickets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Bob creates a ticket
	body = `{"title":"Bob ticket"}`
	req = httptest.NewRequest("POST", "/api/tickets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenBob)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Alice lists mine
	req = httptest.NewRequest("GET", "/api/tickets?mine=true", nil)
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var tickets []model.Ticket
	json.NewDecoder(w.Body).Decode(&tickets)
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket for alice, got %d", len(tickets))
	}
	if tickets[0].Title != "Alice ticket" {
		t.Fatalf("expected Alice ticket, got %q", tickets[0].Title)
	}
}
