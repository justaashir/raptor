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
		if r.URL.Path != "/api/tickets" {
			t.Fatalf("expected /api/tickets, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(model.Ticket{ID: "abc12345", Title: "Test"})
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "")
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

	c := NewClient(ts.URL, "")
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

	c := NewClient(ts.URL, "")
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

	c := NewClient(ts.URL, "")
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

	c := NewClient(ts.URL, "")
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

	c := NewClient(ts.URL, "test-token")
	_, err := c.ListTickets("", false)
	if err != nil {
		t.Fatal(err)
	}
}
