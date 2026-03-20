package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"raptor/model"
	"strings"
)

type Server struct {
	db  *DB
	hub *Hub
	mux *http.ServeMux
}

func NewServer(db *DB, hub *Hub) *Server {
	s := &Server{db: db, hub: hub, mux: http.NewServeMux()}
	s.mux.HandleFunc("/api/tickets", s.handleTickets)
	s.mux.HandleFunc("/api/tickets/", s.handleTicket)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleTickets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		tickets, err := s.db.ListTickets(status)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if tickets == nil {
			tickets = []model.Ticket{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tickets)

	case http.MethodPost:
		var input struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ticket := model.NewTicket(input.Title, input.Content)
		if err := s.db.CreateTicket(ticket); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.hub.Broadcast([]byte(`{"event":"ticket_changed"}`))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ticket)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTicket(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/tickets/")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		ticket, err := s.db.GetTicket(id)
		if err == sql.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ticket)

	case http.MethodPatch:
		var fields map[string]any
		if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.db.UpdateTicket(id, fields); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.hub.Broadcast([]byte(`{"event":"ticket_changed"}`))
		ticket, _ := s.db.GetTicket(id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ticket)

	case http.MethodDelete:
		if err := s.db.DeleteTicket(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.hub.Broadcast([]byte(`{"event":"ticket_changed"}`))
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
