package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"raptor/model"
	"strings"

	"nhooyr.io/websocket"
)

type Server struct {
	db           *DB
	hub          *Hub
	mux          *http.ServeMux
	secret       string
	allowedUsers []string
}

type Option func(*Server)

func WithSecret(secret string) Option {
	return func(s *Server) { s.secret = secret }
}

func WithAllowedUsers(users []string) Option {
	return func(s *Server) { s.allowedUsers = users }
}

func NewServer(db *DB, hub *Hub, opts ...Option) *Server {
	s := &Server{db: db, hub: hub, mux: http.NewServeMux()}
	for _, o := range opts {
		o(s)
	}
	s.mux.HandleFunc("/api/tickets", s.handleTickets)
	s.mux.HandleFunc("/api/tickets/", s.handleTicket)
	s.mux.HandleFunc("/api/version", s.handleVersion)
	s.mux.HandleFunc("/api/auth", s.handleAuth)
	s.mux.HandleFunc("/releases/", s.handleRelease)
	s.mux.HandleFunc("/install.sh", s.handleInstallScript)
	s.mux.HandleFunc("/admin/releases/", s.handleUploadRelease)
	s.mux.HandleFunc("/ws", s.handleWS)
	return s
}

func (s *Server) isAllowedUser(username string) bool {
	if len(s.allowedUsers) == 0 {
		return true
	}
	for _, u := range s.allowedUsers {
		if strings.EqualFold(u, username) {
			return true
		}
	}
	return false
}

type wsConn struct {
	conn *websocket.Conn
	ctx  context.Context
}

func (w *wsConn) Send(msg []byte) error {
	return w.conn.Write(w.ctx, websocket.MessageText, msg)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	wc := &wsConn{conn: c, ctx: r.Context()}
	s.hub.Register(wc)
	defer s.hub.Unregister(wc)

	// Keep connection alive by reading (blocks until client disconnects)
	for {
		_, _, err := c.Read(r.Context())
		if err != nil {
			return
		}
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.authMiddleware(s.mux).ServeHTTP(w, r)
}

func (s *Server) handleTickets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		mine := r.URL.Query().Get("mine")
		username := UsernameFromContext(r.Context())
		tickets, err := s.db.ListTickets(status)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if mine == "true" && username != "" {
			var filtered []model.Ticket
			for _, t := range tickets {
				if t.CreatedBy == username || t.Assignee == username {
					filtered = append(filtered, t)
				}
			}
			tickets = filtered
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
			Assign  string `json:"assignee"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		username := UsernameFromContext(r.Context())
		ticket := model.NewTicket(input.Title, input.Content, username)
		if input.Assign != "" {
			ticket.Assignee = input.Assign
		}
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
