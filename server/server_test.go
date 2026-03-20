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

	ticket := model.NewTicket("Get me", "content")
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

	ticket := model.NewTicket("Original", "")
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

	ticket := model.NewTicket("To delete", "")
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
