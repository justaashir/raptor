package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"raptor/model"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
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
	s.mux.HandleFunc("/api/workspaces/", s.handleWorkspaces)
	s.mux.HandleFunc("/api/version", s.handleVersion)
	s.mux.HandleFunc("/api/auth", s.handleAuth)
	s.mux.HandleFunc("/install.sh", s.handleInstallScript)
	s.mux.HandleFunc("/ws", s.handleWS)
	return s
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

// authorize checks if the user has at least minRole in the workspace.
// For board access with "member" role, also checks board membership.
func (s *Server) authorize(r *http.Request, workspaceID, boardID, minRole string) error {
	username := UsernameFromContext(r.Context())
	if username == "" {
		return fmt.Errorf("unauthorized")
	}
	role, err := s.db.GetMemberRole(workspaceID, username)
	if err != nil {
		return fmt.Errorf("not a workspace member")
	}

	roleLevel := map[string]int{"owner": 3, "admin": 2, "member": 1}
	if roleLevel[role] < roleLevel[minRole] {
		return fmt.Errorf("insufficient permissions")
	}

	if boardID != "" && role == "member" {
		isMember, _ := s.db.IsBoardMember(boardID, username)
		if !isMember {
			return fmt.Errorf("no access to this board")
		}
	}
	return nil
}

// handleWorkspaces dispatches all /api/workspaces/... routes
func (s *Server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/workspaces/")
	path = strings.TrimSuffix(path, "/")
	var parts []string
	if path != "" {
		parts = strings.Split(path, "/")
	}

	switch len(parts) {
	case 0:
		s.handleWorkspaceRoot(w, r)
	case 1:
		s.handleWorkspaceByID(w, r, parts[0])
	case 2:
		switch parts[1] {
		case "members":
			s.handleWorkspaceMembers(w, r, parts[0])
		case "boards":
			s.handleBoards(w, r, parts[0])
		default:
			http.NotFound(w, r)
		}
	case 3:
		switch parts[1] {
		case "members":
			s.handleWorkspaceMember(w, r, parts[0], parts[2])
		case "boards":
			s.handleBoardByID(w, r, parts[0], parts[2])
		default:
			http.NotFound(w, r)
		}
	case 4:
		if parts[1] == "boards" {
			switch parts[3] {
			case "members":
				s.handleBoardMembers(w, r, parts[0], parts[2])
			case "tickets":
				s.handleBoardTickets(w, r, parts[0], parts[2])
			default:
				http.NotFound(w, r)
			}
		} else {
			http.NotFound(w, r)
		}
	case 5:
		if parts[1] == "boards" {
			switch parts[3] {
			case "members":
				s.handleBoardMember(w, r, parts[0], parts[2], parts[4])
			case "tickets":
				s.handleBoardTicket(w, r, parts[0], parts[2], parts[4])
			default:
				http.NotFound(w, r)
			}
		} else {
			http.NotFound(w, r)
		}
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleWorkspaceRoot(w http.ResponseWriter, r *http.Request) {
	username := UsernameFromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		workspaces, err := s.db.ListWorkspacesForUser(username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if workspaces == nil {
			workspaces = []model.Workspace{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(workspaces)

	case http.MethodPost:
		var input struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if input.Name == "" {
			http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
			return
		}
		id := uuid.New().String()[:8]
		if err := s.db.CreateWorkspace(id, input.Name, username); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(model.Workspace{ID: id, Name: input.Name, CreatedBy: username})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWorkspaceByID(w http.ResponseWriter, r *http.Request, wid string) {
	switch r.Method {
	case http.MethodDelete:
		if err := s.authorize(r, wid, "", "owner"); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusForbidden)
			return
		}
		if err := s.db.DeleteWorkspace(wid); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWorkspaceMembers(w http.ResponseWriter, r *http.Request, wid string) {
	switch r.Method {
	case http.MethodGet:
		if err := s.authorize(r, wid, "", "member"); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusForbidden)
			return
		}
		members, err := s.db.ListWorkspaceMembers(wid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if members == nil {
			members = []model.WorkspaceMember{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(members)

	case http.MethodPost:
		if err := s.authorize(r, wid, "", "admin"); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusForbidden)
			return
		}
		var input struct {
			Username string `json:"username"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if input.Role == "" {
			input.Role = "member"
		}
		if !model.ValidRole(input.Role) {
			http.Error(w, `{"error":"invalid role"}`, http.StatusBadRequest)
			return
		}
		if err := s.db.AddWorkspaceMember(wid, input.Username, input.Role); err != nil {
			if errors.Is(err, ErrAlreadyMember) {
				http.Error(w, `{"error":"user is already a member of this workspace"}`, http.StatusConflict)
				return
			}
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"username": input.Username, "role": input.Role})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWorkspaceMember(w http.ResponseWriter, r *http.Request, wid, username string) {
	switch r.Method {
	case http.MethodPatch:
		if err := s.authorize(r, wid, "", "owner"); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusForbidden)
			return
		}
		var input struct {
			Role string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !model.ValidRole(input.Role) {
			http.Error(w, `{"error":"invalid role"}`, http.StatusBadRequest)
			return
		}
		if err := s.db.UpdateMemberRole(wid, username, input.Role); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"username": username, "role": input.Role})

	case http.MethodDelete:
		if err := s.authorize(r, wid, "", "admin"); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusForbidden)
			return
		}
		if err := s.db.RemoveWorkspaceMember(wid, username); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleBoards(w http.ResponseWriter, r *http.Request, wid string) {
	username := UsernameFromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		if err := s.authorize(r, wid, "", "member"); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusForbidden)
			return
		}
		boards, err := s.db.ListBoardsForUser(wid, username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if boards == nil {
			boards = []model.Board{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(boards)

	case http.MethodPost:
		if err := s.authorize(r, wid, "", "admin"); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusForbidden)
			return
		}
		var input struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if input.Name == "" {
			http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
			return
		}
		id := uuid.New().String()[:8]
		if err := s.db.CreateBoard(id, wid, input.Name, username); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(model.Board{ID: id, WorkspaceID: wid, Name: input.Name, CreatedBy: username})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleBoardByID(w http.ResponseWriter, r *http.Request, wid, bid string) {
	switch r.Method {
	case http.MethodDelete:
		if err := s.authorize(r, wid, "", "admin"); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusForbidden)
			return
		}
		if err := s.db.DeleteBoard(bid); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleBoardMembers(w http.ResponseWriter, r *http.Request, wid, bid string) {
	switch r.Method {
	case http.MethodGet:
		if err := s.authorize(r, wid, bid, "member"); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusForbidden)
			return
		}
		members, err := s.db.ListBoardMembers(bid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if members == nil {
			members = []model.BoardMember{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(members)

	case http.MethodPost:
		if err := s.authorize(r, wid, "", "admin"); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusForbidden)
			return
		}
		var input struct {
			Username string `json:"username"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.db.AddBoardMember(bid, input.Username); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"username": input.Username, "board_id": bid})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleBoardMember(w http.ResponseWriter, r *http.Request, wid, bid, username string) {
	switch r.Method {
	case http.MethodDelete:
		if err := s.authorize(r, wid, "", "admin"); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusForbidden)
			return
		}
		if err := s.db.RemoveBoardMember(bid, username); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleBoardTickets(w http.ResponseWriter, r *http.Request, wid, bid string) {
	if err := s.authorize(r, wid, bid, "member"); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusForbidden)
		return
	}
	username := UsernameFromContext(r.Context())

	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		mine := r.URL.Query().Get("mine")
		tickets, err := s.db.ListTickets(bid, status)
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
		if input.Assign != "" {
			isMember, _ := s.db.IsBoardMember(bid, input.Assign)
			if !isMember {
				http.Error(w, `{"error":"assignee is not a board member"}`, http.StatusBadRequest)
				return
			}
		}
		ticket := model.NewTicket(input.Title, input.Content, username)
		ticket.BoardID = bid
		if input.Assign != "" {
			ticket.Assignee = input.Assign
			ticket.AssignedBy = username
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

func (s *Server) handleBoardTicket(w http.ResponseWriter, r *http.Request, wid, bid, tid string) {
	if err := s.authorize(r, wid, bid, "member"); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		ticket, err := s.db.GetTicket(tid)
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
		if assignee, ok := fields["assignee"].(string); ok && assignee != "" {
			isMember, _ := s.db.IsBoardMember(bid, assignee)
			if !isMember {
				http.Error(w, `{"error":"assignee is not a board member"}`, http.StatusBadRequest)
				return
			}
			fields["assigned_by"] = UsernameFromContext(r.Context())
		}
		if err := s.db.UpdateTicket(tid, fields); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.hub.Broadcast([]byte(`{"event":"ticket_changed"}`))
		ticket, _ := s.db.GetTicket(tid)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ticket)

	case http.MethodDelete:
		if err := s.db.DeleteTicket(tid); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.hub.Broadcast([]byte(`{"event":"ticket_changed"}`))
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
