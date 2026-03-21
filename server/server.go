package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"raptor/model"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
	"nhooyr.io/websocket"
)

var allowedPatchFields = map[string]bool{
	"title": true, "content": true, "status": true,
	"assignee": true,
}

type Server struct {
	db           *DB
	hub          *Hub
	Echo         *echo.Echo
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
	s := &Server{db: db, hub: hub, Echo: echo.New()}
	s.Echo.HideBanner = true
	s.Echo.HidePort = true
	s.Echo.Use(middleware.BodyLimit("1M"))
	for _, o := range opts {
		o(s)
	}

	// Public routes (no auth)
	s.Echo.POST("/api/auth", s.handleAuth)
	s.Echo.GET("/api/version", s.handleVersion)
	s.Echo.GET("/api/skill", s.handleSkill)
	s.Echo.GET("/install.sh", s.handleInstallScript)
	s.Echo.GET("/ws", s.handleWS)

	// Authenticated routes
	api := s.Echo.Group("/api", s.authMiddleware)

	// Workspaces
	ws := api.Group("/workspaces")
	ws.GET("/", s.listWorkspaces)
	ws.POST("/", s.createWorkspace)
	ws.DELETE("/:wid", s.deleteWorkspace)

	// Workspace members
	ws.GET("/:wid/members", s.listWorkspaceMembers)
	ws.POST("/:wid/members", s.addWorkspaceMember)
	ws.DELETE("/:wid/members/:username", s.removeWorkspaceMember)

	// Boards
	ws.GET("/:wid/boards", s.listBoards)
	ws.POST("/:wid/boards", s.createBoard)
	ws.GET("/:wid/boards/:bid", s.getBoard)
	ws.PATCH("/:wid/boards/:bid", s.updateBoard)
	ws.DELETE("/:wid/boards/:bid", s.deleteBoard)

	// Tickets
	ws.GET("/:wid/boards/:bid/tickets", s.listTickets)
	ws.POST("/:wid/boards/:bid/tickets", s.createTicket)
	ws.GET("/:wid/boards/:bid/tickets/:tid", s.getTicket)
	ws.PATCH("/:wid/boards/:bid/tickets/:tid", s.updateTicket)
	ws.DELETE("/:wid/boards/:bid/tickets/:tid", s.deleteTicket)

	return s
}

// ServeHTTP implements http.Handler so existing tests and serve.go work.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Echo.ServeHTTP(w, r)
}

// --- helpers ---

func jsonErr(c echo.Context, code int, msg string) error {
	return c.JSON(code, map[string]string{"error": msg})
}

func validateTicketFields(title, content string) string {
	if len(title) > 500 {
		return "title too long (max 500 characters)"
	}
	if len(content) > 100000 {
		return "content too long (max 100000 characters)"
	}
	return ""
}

func username(c echo.Context) string {
	if u, ok := c.Get("username").(string); ok {
		return u
	}
	return ""
}

// authorize checks if the user is a workspace member with at least minRole.
func (s *Server) authorize(c echo.Context, workspaceID, minRole string) error {
	u := username(c)
	if u == "" {
		return errors.New("forbidden")
	}
	role, err := s.db.GetMemberRole(workspaceID, u)
	if err != nil {
		return errors.New("forbidden")
	}
	if roleLevels[role] < roleLevels[minRole] {
		return errors.New("forbidden")
	}
	return nil
}

var roleLevels = map[string]int{"owner": 2, "member": 1}

func genID() string {
	return uuid.New().String()[:12]
}

var errHandled = errors.New("handled")

func (s *Server) requireBoard(c echo.Context) (model.Board, error) {
	wid := c.Param("wid")
	bid := c.Param("bid")
	board, err := s.db.GetBoard(bid)
	if err != nil {
		jsonErr(c, http.StatusNotFound, "board not found")
		return model.Board{}, errHandled
	}
	if board.WorkspaceID != wid {
		jsonErr(c, http.StatusNotFound, "board not found")
		return model.Board{}, errHandled
	}
	return board, nil
}

// --- WebSocket (stays raw, Echo doesn't wrap WS well) ---

type wsConn struct {
	conn *websocket.Conn
	ctx  context.Context
}

func (w *wsConn) Send(msg []byte) error {
	return w.conn.Write(w.ctx, websocket.MessageText, msg)
}

func (s *Server) handleWS(c echo.Context) error {
	ws, err := websocket.Accept(c.Response(), c.Request(), &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil
	}
	defer ws.Close(websocket.StatusNormalClosure, "")

	wc := &wsConn{conn: ws, ctx: c.Request().Context()}
	s.hub.Register(wc)
	defer s.hub.Unregister(wc)

	for {
		_, _, err := ws.Read(c.Request().Context())
		if err != nil {
			return nil
		}
	}
}

// --- Workspace handlers ---

func (s *Server) listWorkspaces(c echo.Context) error {
	workspaces, err := s.db.ListWorkspacesForUser(username(c))
	if err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	if workspaces == nil {
		workspaces = []model.Workspace{}
	}
	return c.JSON(http.StatusOK, workspaces)
}

func (s *Server) createWorkspace(c echo.Context) error {
	var input struct {
		Name string `json:"name"`
	}
	if err := c.Bind(&input); err != nil {
		return jsonErr(c, http.StatusBadRequest, "bad request")
	}
	if input.Name == "" {
		return jsonErr(c, http.StatusBadRequest, "name required")
	}
	id := genID()
	u := username(c)
	if err := s.db.CreateWorkspace(id, input.Name, u); err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	return c.JSON(http.StatusCreated, model.Workspace{ID: id, Name: input.Name, CreatedBy: u})
}

func (s *Server) deleteWorkspace(c echo.Context) error {
	wid := c.Param("wid")
	if err := s.authorize(c, wid, "owner"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}
	if err := s.db.DeleteWorkspace(wid); err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	return c.NoContent(http.StatusNoContent)
}

// --- Workspace member handlers ---

func (s *Server) listWorkspaceMembers(c echo.Context) error {
	wid := c.Param("wid")
	if err := s.authorize(c, wid, "member"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}
	members, err := s.db.ListWorkspaceMembers(wid)
	if err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	if members == nil {
		members = []model.WorkspaceMember{}
	}
	return c.JSON(http.StatusOK, members)
}

func (s *Server) addWorkspaceMember(c echo.Context) error {
	wid := c.Param("wid")
	if err := s.authorize(c, wid, "owner"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}
	var input struct {
		Username string `json:"username"`
	}
	if err := c.Bind(&input); err != nil {
		return jsonErr(c, http.StatusBadRequest, "bad request")
	}
	if err := s.db.AddWorkspaceMember(wid, input.Username, "member"); err != nil {
		if errors.Is(err, ErrAlreadyMember) {
			return jsonErr(c, http.StatusConflict, "user is already a member of this workspace")
		}
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	return c.JSON(http.StatusCreated, map[string]string{"username": input.Username, "role": "member"})
}

func (s *Server) removeWorkspaceMember(c echo.Context) error {
	wid := c.Param("wid")
	user := c.Param("username")
	if err := s.authorize(c, wid, "owner"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}
	// Prevent removing the last owner
	role, err := s.db.GetMemberRole(wid, user)
	if err != nil {
		return jsonErr(c, http.StatusNotFound, "member not found")
	}
	if role == "owner" {
		count, err := s.db.CountOwners(wid)
		if err != nil {
			return jsonErr(c, http.StatusInternalServerError, "internal server error")
		}
		if count <= 1 {
			return jsonErr(c, http.StatusBadRequest, "cannot remove the last owner")
		}
	}
	if err := s.db.RemoveWorkspaceMember(wid, user); err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	return c.NoContent(http.StatusNoContent)
}

// --- Board handlers ---

func (s *Server) listBoards(c echo.Context) error {
	wid := c.Param("wid")
	if err := s.authorize(c, wid, "member"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}
	boards, err := s.db.ListBoardsForUser(wid, username(c))
	if err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	if boards == nil {
		boards = []model.Board{}
	}
	return c.JSON(http.StatusOK, boards)
}

func (s *Server) getBoard(c echo.Context) error {
	wid := c.Param("wid")
	if err := s.authorize(c, wid, "member"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}
	board, err := s.requireBoard(c)
	if err != nil {
		return nil
	}
	return c.JSON(http.StatusOK, board)
}

func (s *Server) createBoard(c echo.Context) error {
	wid := c.Param("wid")
	if err := s.authorize(c, wid, "owner"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}
	var input struct {
		Name     string   `json:"name"`
		Statuses []string `json:"statuses"`
	}
	if err := c.Bind(&input); err != nil {
		return jsonErr(c, http.StatusBadRequest, "bad request")
	}
	if input.Name == "" {
		return jsonErr(c, http.StatusBadRequest, "name required")
	}
	if len(input.Statuses) == 0 {
		input.Statuses = model.DefaultStatuses
	}
	for _, st := range input.Statuses {
		if st == "" || strings.ContainsAny(st, ", ") {
			return jsonErr(c, http.StatusBadRequest, "invalid status name: must be non-empty with no commas or spaces")
		}
	}
	id := genID()
	u := username(c)
	if err := s.db.CreateBoard(id, wid, input.Name, u, input.Statuses); err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	return c.JSON(http.StatusCreated, model.Board{
		ID: id, WorkspaceID: wid, Name: input.Name,
		Statuses: strings.Join(input.Statuses, ","), CreatedBy: u,
	})
}

func (s *Server) updateBoard(c echo.Context) error {
	wid := c.Param("wid")
	bid := c.Param("bid")
	if err := s.authorize(c, wid, "owner"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}
	_, err := s.requireBoard(c)
	if err != nil {
		return nil
	}
	var input struct {
		Name     *string  `json:"name"`
		Statuses []string `json:"statuses"`
	}
	if err := c.Bind(&input); err != nil {
		return jsonErr(c, http.StatusBadRequest, "bad request")
	}
	fields := map[string]any{}
	if input.Name != nil && *input.Name != "" {
		fields["name"] = *input.Name
	}
	if len(input.Statuses) > 0 {
		for _, st := range input.Statuses {
			if st == "" || strings.ContainsAny(st, ", ") {
				return jsonErr(c, http.StatusBadRequest, "invalid status name: must be non-empty with no commas or spaces")
			}
		}
		fields["statuses"] = strings.Join(input.Statuses, ",")
	}
	if len(fields) == 0 {
		return jsonErr(c, http.StatusBadRequest, "no valid fields")
	}
	if err := s.db.UpdateBoard(bid, fields); err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	updated, err := s.db.GetBoard(bid)
	if err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	return c.JSON(http.StatusOK, updated)
}

func (s *Server) deleteBoard(c echo.Context) error {
	wid := c.Param("wid")
	bid := c.Param("bid")
	if err := s.authorize(c, wid, "owner"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}
	if _, err := s.requireBoard(c); err != nil {
		return nil
	}
	if err := s.db.DeleteBoard(bid); err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	return c.NoContent(http.StatusNoContent)
}

// --- Ticket handlers ---

func (s *Server) listTickets(c echo.Context) error {
	wid := c.Param("wid")
	bid := c.Param("bid")
	if err := s.authorize(c, wid, "member"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}
	u := username(c)

	// Stats mode
	if c.QueryParam("stats") == "true" {
		counts, err := s.db.TicketStats(bid)
		if err != nil {
			return jsonErr(c, http.StatusInternalServerError, "internal server error")
		}
		total := 0
		for _, cnt := range counts {
			total += cnt
		}
		return c.JSON(http.StatusOK, map[string]any{"total": total, "counts": counts})
	}

	query := c.QueryParam("q")
	status := c.QueryParam("status")
	mine := c.QueryParam("mine")

	var tickets []model.Ticket
	var err error
	if query != "" {
		tickets, err = s.db.SearchTickets(bid, query)
	} else {
		tickets, err = s.db.ListTickets(bid, status)
	}
	if err != nil {
		log.Printf("ticket list error: %v", err)
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	if mine == "true" && u != "" {
		var filtered []model.Ticket
		for _, t := range tickets {
			if t.CreatedBy == u || t.Assignee == u {
				filtered = append(filtered, t)
			}
		}
		tickets = filtered
	}
	if tickets == nil {
		tickets = []model.Ticket{}
	}
	return c.JSON(http.StatusOK, tickets)
}

func (s *Server) createTicket(c echo.Context) error {
	wid := c.Param("wid")
	bid := c.Param("bid")
	if err := s.authorize(c, wid, "member"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}
	u := username(c)

	var input struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Assign  string `json:"assignee"`
	}
	if err := c.Bind(&input); err != nil {
		return jsonErr(c, http.StatusBadRequest, "bad request")
	}
	if msg := validateTicketFields(input.Title, input.Content); msg != "" {
		return jsonErr(c, http.StatusBadRequest, msg)
	}
	ticket := model.NewTicket(input.Title, input.Content, u)
	ticket.BoardID = bid
	// Set default status to the board's first status
	board, err := s.db.GetBoard(bid)
	if err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	if board.WorkspaceID != wid {
		return jsonErr(c, http.StatusNotFound, "board not found")
	}
	statuses := board.StatusList()
	if len(statuses) > 0 {
		ticket.Status = model.Status(statuses[0])
	}
	if input.Assign != "" {
		if _, err := s.db.GetMemberRole(wid, input.Assign); err != nil {
			return jsonErr(c, http.StatusBadRequest, "assignee is not a workspace member")
		}
		ticket.Assignee = input.Assign
		ticket.AssignedBy = u
	}
	if err := s.db.CreateTicket(ticket); err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	s.hub.Broadcast([]byte(`{"event":"ticket_changed"}`))
	return c.JSON(http.StatusCreated, ticket)
}

func (s *Server) getTicket(c echo.Context) error {
	wid := c.Param("wid")
	bid := c.Param("bid")
	tid := c.Param("tid")
	if err := s.authorize(c, wid, "member"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}
	ticket, err := s.db.GetTicket(tid)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return jsonErr(c, http.StatusNotFound, "not found")
	}
	if err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	if ticket.BoardID != bid {
		return jsonErr(c, http.StatusNotFound, "not found")
	}
	return c.JSON(http.StatusOK, ticket)
}

func (s *Server) updateTicket(c echo.Context) error {
	wid := c.Param("wid")
	bid := c.Param("bid")
	tid := c.Param("tid")
	if err := s.authorize(c, wid, "member"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}

	// Verify ticket exists and belongs to this board
	existing, err := s.db.GetTicket(tid)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return jsonErr(c, http.StatusNotFound, "not found")
	}
	if err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	if existing.BoardID != bid {
		return jsonErr(c, http.StatusNotFound, "not found")
	}

	var fields map[string]any
	if err := json.NewDecoder(c.Request().Body).Decode(&fields); err != nil {
		return c.String(http.StatusBadRequest, "invalid request body")
	}
	// Whitelist allowed fields
	for k := range fields {
		if !allowedPatchFields[k] {
			delete(fields, k)
		}
	}
	if len(fields) == 0 {
		return jsonErr(c, http.StatusBadRequest, "no valid fields")
	}
	// Validate title/content length
	title, _ := fields["title"].(string)
	content, _ := fields["content"].(string)
	if msg := validateTicketFields(title, content); msg != "" {
		return jsonErr(c, http.StatusBadRequest, msg)
	}
	// Validate status against board's allowed statuses
	if st, ok := fields["status"].(string); ok {
		board, err := s.db.GetBoard(bid)
		if err != nil {
			return jsonErr(c, http.StatusInternalServerError, "internal server error")
		}
		if board.WorkspaceID != wid {
			return jsonErr(c, http.StatusNotFound, "board not found")
		}
		if !board.ValidStatus(st) {
			return jsonErr(c, http.StatusBadRequest, "invalid status")
		}
	}
	if assignee, ok := fields["assignee"].(string); ok && assignee != "" {
		if _, err := s.db.GetMemberRole(wid, assignee); err != nil {
			return jsonErr(c, http.StatusBadRequest, "assignee is not a workspace member")
		}
		fields["assigned_by"] = username(c)
	}
	if err := s.db.UpdateTicket(tid, fields); err != nil {
		log.Printf("ticket update error: %v", err)
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	s.hub.Broadcast([]byte(`{"event":"ticket_changed"}`))
	ticket, err := s.db.GetTicket(tid)
	if err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	return c.JSON(http.StatusOK, ticket)
}

func (s *Server) deleteTicket(c echo.Context) error {
	wid := c.Param("wid")
	bid := c.Param("bid")
	tid := c.Param("tid")
	if err := s.authorize(c, wid, "member"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}
	ticket, err := s.db.GetTicket(tid)
	if err != nil {
		return jsonErr(c, http.StatusNotFound, "not found")
	}
	if ticket.BoardID != bid {
		return jsonErr(c, http.StatusNotFound, "not found")
	}
	if err := s.db.DeleteTicket(tid); err != nil {
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	s.hub.Broadcast([]byte(`{"event":"ticket_changed"}`))
	return c.NoContent(http.StatusNoContent)
}
