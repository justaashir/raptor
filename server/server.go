package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"raptor/model"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
	"nhooyr.io/websocket"
)

type rateLimitEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rateLimitEntry
	r        rate.Limit
	burst    int
}

func newIPRateLimiter(r rate.Limit, burst int) *ipRateLimiter {
	rl := &ipRateLimiter{
		limiters: make(map[string]*rateLimitEntry),
		r:        r,
		burst:    burst,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *ipRateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	entry, ok := rl.limiters[ip]
	if !ok {
		entry = &rateLimitEntry{limiter: rate.NewLimiter(rl.r, rl.burst)}
		rl.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	rl.mu.Unlock()
	return entry.limiter.Allow()
}

func (rl *ipRateLimiter) cleanup(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	for ip, entry := range rl.limiters {
		if entry.lastSeen.Before(cutoff) {
			delete(rl.limiters, ip)
		}
	}
}

func (rl *ipRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.cleanup(time.Hour)
	}
}

var ticketChangedEvent = []byte(`{"event":"ticket_changed"}`)

var allowedPatchFields = map[string]bool{
	"title": true, "content": true, "status": true,
	"assignee": true,
}

type Server struct {
	db           *DB
	hub          *Hub
	Echo         *echo.Echo
	secret       string
	allowedUsers map[string]bool
}

type Option func(*Server)

func WithSecret(secret string) Option {
	return func(s *Server) { s.secret = secret }
}

func WithAllowedUsers(users []string) Option {
	return func(s *Server) {
		s.allowedUsers = make(map[string]bool, len(users))
		for _, u := range users {
			s.allowedUsers[strings.ToLower(u)] = true
		}
	}
}

func NewServer(db *DB, hub *Hub, opts ...Option) *Server {
	s := &Server{db: db, hub: hub, Echo: echo.New()}
	s.Echo.HideBanner = true
	s.Echo.HidePort = true
	for _, o := range opts {
		o(s)
	}

	// Security headers middleware
	s.Echo.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			return next(c)
		}
	})

	// Public routes (no auth)
	authLimiter := newIPRateLimiter(rate.Every(time.Second), 5)
	s.Echo.POST("/api/auth", func(c echo.Context) error {
		if !authLimiter.allow(c.RealIP()) {
			return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
		}
		return s.handleAuth(c)
	})
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
	return model.GenID()
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
	ctx, cancel := context.WithTimeout(w.ctx, 10*time.Second)
	defer cancel()
	return w.conn.Write(ctx, websocket.MessageText, msg)
}

func (s *Server) handleWS(c echo.Context) error {
	if s.secret == "" {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "server not configured"})
	}
	tokenStr := c.QueryParam("token")
	if tokenStr == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing token"})
	}
	if _, err := ValidateToken(tokenStr, s.secret); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
	}

	// OriginPatterns: ["*"] is intentional — the WS endpoint is auth-gated via
	// JWT token in the query string, so origin checking is redundant. Tightening
	// to specific origins would break CLI clients connecting from arbitrary hosts.
	ws, err := websocket.Accept(c.Response(), c.Request(), &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
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
	if len(input.Name) > 100 {
		return jsonErr(c, http.StatusBadRequest, "name too long (max 100 characters)")
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
	if !validUsername.MatchString(input.Username) || len(input.Username) > 39 {
		return jsonErr(c, http.StatusBadRequest, "invalid username")
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

func validateStatuses(statuses []string) error {
	for _, st := range statuses {
		if st == "" || strings.ContainsAny(st, ", ") {
			return fmt.Errorf("invalid status name: must be non-empty with no commas or spaces")
		}
	}
	return nil
}

// --- Board handlers ---

func (s *Server) listBoards(c echo.Context) error {
	wid := c.Param("wid")
	if err := s.authorize(c, wid, "member"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}
	boards, err := s.db.ListBoardsForUser(wid)
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
	if len(input.Name) > 100 {
		return jsonErr(c, http.StatusBadRequest, "name too long (max 100 characters)")
	}
	if len(input.Statuses) == 0 {
		input.Statuses = model.DefaultStatuses
	}
	if err := validateStatuses(input.Statuses); err != nil {
		return jsonErr(c, http.StatusBadRequest, err.Error())
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
		if err := validateStatuses(input.Statuses); err != nil {
			return jsonErr(c, http.StatusBadRequest, err.Error())
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
	} else if mine == "true" && u != "" {
		tickets, err = s.db.ListTicketsMine(bid, u)
	} else {
		tickets, err = s.db.ListTickets(bid, status)
	}
	if err != nil {
		log.Printf("ticket list error: %v", err)
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
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
	if input.Title == "" {
		return jsonErr(c, http.StatusBadRequest, "title required")
	}
	ticket := model.NewTicket(input.Title, input.Content, u)
	ticket.BoardID = bid
	// Set default status to the board's first status
	board, err := s.requireBoard(c)
	if err != nil {
		return nil
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
	s.hub.Broadcast(ticketChangedEvent)
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
	ticket, err := s.db.UpdateTicket(tid, fields)
	if err != nil {
		log.Printf("ticket update error: %v", err)
		return jsonErr(c, http.StatusInternalServerError, "internal server error")
	}
	s.hub.Broadcast(ticketChangedEvent)
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
	s.hub.Broadcast(ticketChangedEvent)
	return c.NoContent(http.StatusNoContent)
}
