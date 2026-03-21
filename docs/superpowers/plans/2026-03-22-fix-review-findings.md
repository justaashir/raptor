# Fix All Code Review Findings — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Address all actionable findings from the 5-agent code review (security hardening, performance fixes, code quality, architecture cleanup, test coverage) without rewriting the auth model.

**Architecture:** Fixes are grouped by domain and ordered so earlier tasks don't conflict with later ones. Security and performance fixes come first, then DRY/cleanup, then tests (which validate everything).

**Tech Stack:** Go, Echo v4, SQLite/GORM, nhooyr.io/websocket, Bubble Tea, lipgloss, cobra, resty

---

## File Map

| File | Changes |
|------|---------|
| `server/hub.go` | Per-client buffered send channels |
| `server/server.go` | WS auth, CORS/security headers, rate limiting, status validation extract, input validation, reduce DB calls |
| `server/auth.go` | Handle IsWorkspaceMember error, username validation on invite |
| `server/db.go` | MaxOpenConns(4), mine filter SQL, DeleteWorkspace single query |
| `server/releases.go` | Require SERVER_BASE_URL in production |
| `tui/app.go` | Keep WS alive, debounce fetches |
| `tui/list_pane.go` | O(1) truncation via ansi.Truncate, pre-allocate styles |
| `tui/statusbar.go` | Cache rendered bar |
| `tui/styles.go` | Remove dead styles |
| `tui/detail_pane.go` | Rename sp/bp/up helpers |
| `cmd/update.go` | HTTP client with timeout |
| `cmd/login.go` | HTTP client with timeout |
| `cmd/edit.go` | Use Flags().Changed() for clearing |
| `cmd/board.go` | Extract parseStatuses helper |
| `cmd/config.go` | Handle UserHomeDir error |
| `cmd/list.go` | Use shared status palette |
| `tui/cache.go` | Handle UserHomeDir error |
| `model/shared.go` (new) | Shared githubRepo const, statusPalette |
| Tests: `server/*_test.go`, `client/client_test.go`, `cmd/*_test.go`, `tui/*_test.go` | New tests |

---

### Task 1: Create Feature Branch

**Files:** None (git only)

- [ ] **Step 1: Create and switch to feature branch**

```bash
git checkout -b fix/review-findings
```

- [ ] **Step 2: Verify clean state**

```bash
git status
```

---

### Task 2: WebSocket Authentication

**Files:**
- Modify: `server/server.go:54,155-174`
- Modify: `server/server_test.go`
- Modify: `tui/app.go:568-605`

- [ ] **Step 1: Write failing test — WS rejects unauthenticated connections**

In `server/server_test.go`:
```go
func TestServer_WebSocket_RequiresAuth(t *testing.T) {
	srv := newTestServerWithAuth(t, "testsecret", []string{"alice"})
	ts := httptest.NewServer(srv.Echo)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, _, err := websocket.Dial(ctx, wsURL, nil)
	if err == nil {
		t.Fatal("expected WS connection without token to be rejected")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./server/ -run TestServer_WebSocket_RequiresAuth -v`
Expected: FAIL (currently accepts all connections)

- [ ] **Step 3: Write failing test — WS accepts authenticated connections**

```go
func TestServer_WebSocket_AcceptsAuth(t *testing.T) {
	srv := newTestServerWithAuth(t, "testsecret", []string{"alice"})
	ts := httptest.NewServer(srv.Echo)
	defer ts.Close()

	token, _ := srv.GenerateToken("alice")
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?token=" + token
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("expected authenticated WS to connect: %v", err)
	}
	conn.Close(websocket.StatusNormalClosure, "")
}
```

- [ ] **Step 4: Implement WS auth — validate token query param**

In `server/server.go`, modify `handleWS`:
```go
func (s *Server) handleWS(c echo.Context) error {
	// Validate token from query parameter
	tokenStr := c.QueryParam("token")
	if s.secret == "" {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "server not configured"})
	}
	if tokenStr == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "token required"})
	}
	if _, err := ValidateToken(tokenStr, s.secret); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
	}

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
```

- [ ] **Step 5: Update TUI listenWS to pass token as query param**

In `tui/app.go`, update the WebSocket URL construction in `listenWS`:
```go
wsURL := strings.Replace(a.serverURL, "http", "ws", 1) + "/ws?token=" + a.token
```

- [ ] **Step 6: Run both tests to verify they pass**

Run: `go test ./server/ -run TestServer_WebSocket -v`
Expected: PASS

- [ ] **Step 7: Run all tests to check for regressions**

Run: `go test ./... -count=1`

- [ ] **Step 8: Commit**

```bash
git add server/server.go server/server_test.go tui/app.go
git commit -m "feat: require auth token for WebSocket connections

Validates JWT token passed as query parameter on /ws endpoint.
Removes InsecureSkipVerify, uses OriginPatterns instead."
```

---

### Task 3: CORS + Security Headers

**Files:**
- Modify: `server/server.go:41-48`
- Modify: `server/server_test.go`

- [ ] **Step 1: Write failing test — security headers present**

```go
func TestServer_SecurityHeaders(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	ts := httptest.NewServer(srv.Echo)
	defer ts.Close()

	resp, _ := http.Get(ts.URL + "/api/version")
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options header")
	}
	if resp.Header.Get("X-Frame-Options") != "DENY" {
		t.Error("missing X-Frame-Options header")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

- [ ] **Step 3: Add security headers middleware**

In `server/server.go` `NewServer`, before route registration:
```go
s.Echo.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("X-Content-Type-Options", "nosniff")
		c.Response().Header().Set("X-Frame-Options", "DENY")
		return next(c)
	}
})
```

- [ ] **Step 4: Run test to verify it passes**

- [ ] **Step 5: Commit**

```bash
git add server/server.go server/server_test.go
git commit -m "feat: add security headers middleware (X-Content-Type-Options, X-Frame-Options)"
```

---

### Task 4: Rate Limiting on Auth Endpoint

**Files:**
- Modify: `server/server.go`
- Modify: `server/server_test.go`

- [ ] **Step 1: Write failing test — rate limiting blocks excessive auth attempts**

```go
func TestServer_Auth_RateLimited(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	ts := httptest.NewServer(srv.Echo)
	defer ts.Close()

	body := `{"username":"alice"}`
	var lastStatus int
	for i := 0; i < 20; i++ {
		resp, _ := http.Post(ts.URL+"/api/auth", "application/json", strings.NewReader(body))
		lastStatus = resp.StatusCode
		resp.Body.Close()
	}
	if lastStatus != http.StatusTooManyRequests {
		t.Errorf("expected 429 after rapid auth attempts, got %d", lastStatus)
	}
}
```

- [ ] **Step 2: Implement rate limiting middleware on auth route**

Use Echo's built-in rate limiter or a simple in-memory rate limiter. Add rate limiting specifically to the auth endpoint in `NewServer`:
```go
import "golang.org/x/time/rate"

// In NewServer, wrap the auth handler:
authLimiter := rate.NewLimiter(rate.Every(time.Second), 5) // 5 req/sec burst
s.Echo.POST("/api/auth", func(c echo.Context) error {
	if !authLimiter.Allow() {
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
	}
	return s.handleAuth(c)
})
```

Add `"golang.org/x/time/rate"` and `"time"` to imports. Run `go get golang.org/x/time`.

- [ ] **Step 3: Run tests**

Run: `go test ./server/ -run TestServer_Auth_RateLimited -v`

- [ ] **Step 4: Run all tests**

Run: `go test ./... -count=1`

- [ ] **Step 5: Commit**

```bash
git add server/server.go server/server_test.go go.mod go.sum
git commit -m "feat: add rate limiting on auth endpoint (5 req/sec burst)"
```

---

### Task 5: Hub Broadcast — Per-Client Buffered Channels

**Files:**
- Modify: `server/hub.go`
- Modify: `server/hub_test.go`

- [ ] **Step 1: Write failing test — slow client doesn't block others**

```go
func TestHub_SlowClientDoesNotBlock(t *testing.T) {
	h := NewHub()
	go h.Run()
	defer h.Stop()

	slow := &slowConn{delay: 100 * time.Millisecond}
	fast := &fakeConn{}
	h.Register(slow)
	h.Register(fast)
	time.Sleep(10 * time.Millisecond)

	start := time.Now()
	done := make(chan struct{})
	h.broadcast <- broadcastMsg{data: []byte("test"), done: done}
	<-done
	elapsed := time.Since(start)

	if elapsed > 50*time.Millisecond {
		t.Errorf("broadcast took %v, should not block on slow client", elapsed)
	}
	time.Sleep(10 * time.Millisecond)
	if len(fast.msgs) == 0 {
		t.Error("fast client should have received message")
	}
}

type slowConn struct {
	delay time.Duration
	msgs  []string
}

func (s *slowConn) Send(msg []byte) error {
	time.Sleep(s.delay)
	s.msgs = append(s.msgs, string(msg))
	return nil
}
```

- [ ] **Step 2: Run test to verify it fails**

- [ ] **Step 3: Implement per-client buffered send**

Replace the Hub to use per-client goroutines with buffered channels:
```go
type client struct {
	conn Conn
	send chan []byte
}

type Hub struct {
	register   chan Conn
	unregister chan Conn
	broadcast  chan broadcastMsg
	stop       chan struct{}
	clients    map[Conn]*client
}

func (h *Hub) Run() {
	for {
		select {
		case <-h.stop:
			for _, cl := range h.clients {
				close(cl.send)
			}
			return
		case c := <-h.register:
			cl := &client{conn: c, send: make(chan []byte, 16)}
			h.clients[c] = cl
			go cl.writePump()
		case c := <-h.unregister:
			if cl, ok := h.clients[c]; ok {
				close(cl.send)
				delete(h.clients, c)
			}
		case msg := <-h.broadcast:
			for conn, cl := range h.clients {
				select {
				case cl.send <- msg.data:
				default:
					close(cl.send)
					delete(h.clients, conn)
				}
			}
			if msg.done != nil {
				close(msg.done)
			}
		}
	}
}

func (cl *client) writePump() {
	for msg := range cl.send {
		if err := cl.conn.Send(msg); err != nil {
			return
		}
	}
}
```

- [ ] **Step 4: Run all hub tests**

Run: `go test ./server/ -run TestHub -v`

- [ ] **Step 5: Run all tests**

Run: `go test ./... -count=1`

- [ ] **Step 6: Commit**

```bash
git add server/hub.go server/hub_test.go
git commit -m "perf: use per-client buffered channels in hub broadcast

Prevents slow/dead clients from blocking all broadcasts.
Drops messages for clients that can't keep up."
```

---

### Task 6: WebSocket Keep-Alive + Debounce in TUI

**Files:**
- Modify: `tui/app.go`
- Modify: `tui/app_test.go`

- [ ] **Step 1: Write test for debounce behavior**

```go
func TestApp_WSDebounce(t *testing.T) {
	app := newTestApp()
	// Simulate rapid WS messages
	var cmd tea.Cmd
	for i := 0; i < 5; i++ {
		_, cmd = app.Update(wsMsg{})
	}
	// Should batch into a single fetch, not 5
	// The cmd should be non-nil (debounced fetch pending)
	if cmd == nil {
		t.Error("expected a debounced fetch command")
	}
}
```

- [ ] **Step 2: Implement persistent WS connection**

Refactor `listenWS` to keep the connection alive and send multiple messages through the Bubble Tea event loop without disconnecting:

```go
func (a *App) listenWS() tea.Msg {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		wsURL := strings.Replace(a.serverURL, "http", "ws", 1) + "/ws?token=" + a.token
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		c, _, err := websocket.Dial(ctx, wsURL, nil)
		cancel()
		if err != nil {
			time.Sleep(backoff)
			backoff = min(backoff*2, maxBackoff)
			continue
		}
		backoff = time.Second

		for {
			_, data, err := c.Read(context.Background())
			if err != nil {
				break
			}
			var ev struct{ Event string `json:"event"` }
			if json.Unmarshal(data, &ev) == nil && ev.Event == "ticket_changed" {
				return wsMsg{}
			}
		}
		c.Close(websocket.StatusNormalClosure, "")
	}
}
```

- [ ] **Step 3: Add debounce on wsMsg handling**

In `App.Update`, replace the direct fetch with a debounced timer:

```go
case wsMsg:
	// Debounce: wait 200ms before fetching, coalescing rapid events
	return m, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return debouncedFetchMsg{}
	})

case debouncedFetchMsg:
	return m, a.fetchTickets
```

Add the new message type:
```go
type debouncedFetchMsg struct{}
```

- [ ] **Step 4: Run tests**

Run: `go test ./tui/ -v -count=1`

- [ ] **Step 5: Commit**

```bash
git add tui/app.go tui/app_test.go
git commit -m "perf: keep WebSocket alive, debounce ticket fetches

WS connection persists across events instead of reconnecting per message.
Rapid ticket_changed events are coalesced with 200ms debounce."
```

---

### Task 7: Database Performance — MaxOpenConns, Mine Filter, DeleteWorkspace

**Files:**
- Modify: `server/db.go:65,133-152,216-227`
- Modify: `server/db_test.go`
- Modify: `server/server.go:439-447`

- [ ] **Step 1: Write failing test — ListTicketsMine filters in SQL**

```go
func TestDB_ListTicketsMine(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "Workspace", "alice")
	db.CreateBoard("b1", "ws1", "Board", "alice", model.DefaultStatuses)
	db.CreateTicket(model.Ticket{ID: "t1", BoardID: "b1", Title: "Alice's", CreatedBy: "alice"})
	db.CreateTicket(model.Ticket{ID: "t2", BoardID: "b1", Title: "Bob's", CreatedBy: "bob"})
	db.CreateTicket(model.Ticket{ID: "t3", BoardID: "b1", Title: "Assigned", CreatedBy: "bob", Assignee: "alice"})

	tickets, err := db.ListTicketsMine("b1", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(tickets) != 2 {
		t.Errorf("expected 2 tickets for alice, got %d", len(tickets))
	}
}
```

- [ ] **Step 2: Implement ListTicketsMine**

In `server/db.go`:
```go
func (d *DB) ListTicketsMine(boardID, username string) ([]model.Ticket, error) {
	var tickets []model.Ticket
	err := d.conn.Where("board_id = ? AND (created_by = ? OR assignee = ?)", boardID, username, username).
		Order("created_at desc").Find(&tickets).Error
	return tickets, err
}
```

- [ ] **Step 3: Update MaxOpenConns to 4**

In `server/db.go`, change line 65:
```go
sqlDB.SetMaxOpenConns(4)
```

- [ ] **Step 4: Write test for DeleteWorkspace efficiency**

Verify DeleteWorkspace works correctly (behavior test, not perf):
```go
func TestDB_DeleteWorkspace_CascadesAll(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "WS", "alice")
	db.CreateBoard("b1", "ws1", "Board1", "alice", model.DefaultStatuses)
	db.CreateBoard("b2", "ws1", "Board2", "alice", model.DefaultStatuses)
	db.CreateTicket(model.Ticket{ID: "t1", BoardID: "b1", Title: "T1"})
	db.CreateTicket(model.Ticket{ID: "t2", BoardID: "b2", Title: "T2"})

	err := db.DeleteWorkspace("ws1")
	if err != nil {
		t.Fatal(err)
	}
	// Verify all tickets and boards are gone
	var count int64
	db.conn.Model(&model.Ticket{}).Where("board_id IN ?", []string{"b1", "b2"}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 tickets after cascade delete, got %d", count)
	}
}
```

- [ ] **Step 5: Optimize DeleteWorkspace — single DELETE**

```go
func (d *DB) DeleteWorkspace(id string) error {
	return d.conn.Transaction(func(tx *gorm.DB) error {
		// Delete tickets for all boards in workspace in one query
		if err := tx.Where("board_id IN (?)",
			tx.Model(&model.Board{}).Select("id").Where("workspace_id = ?", id),
		).Delete(&model.Ticket{}).Error; err != nil {
			return err
		}
		if err := tx.Where("workspace_id = ?", id).Delete(&model.Board{}).Error; err != nil {
			return err
		}
		if err := tx.Where("workspace_id = ?", id).Delete(&model.WorkspaceMember{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&model.Workspace{}).Error
	})
}
```

- [ ] **Step 6: Update listTickets handler to use ListTicketsMine**

In `server/server.go`, update the handler that uses `mine` filter to call `db.ListTicketsMine()` instead of filtering in Go.

- [ ] **Step 7: Run all tests**

Run: `go test ./... -count=1`

- [ ] **Step 8: Commit**

```bash
git add server/db.go server/db_test.go server/server.go
git commit -m "perf: increase MaxOpenConns to 4, mine filter in SQL, single-query cascade delete"
```

---

### Task 8: TUI Rendering Performance

**Files:**
- Modify: `tui/list_pane.go:87-153`
- Modify: `tui/list_pane_test.go`
- Modify: `tui/statusbar.go`
- Modify: `tui/statusbar_test.go`

- [ ] **Step 1: Write test for truncateToWidth with ansi.Truncate**

```go
func TestTruncateToWidth(t *testing.T) {
	cases := []struct{ input string; width int; fits bool }{
		{"hello", 10, true},
		{"hello world this is long", 10, false},
		{"", 5, true},
		{"hi", 0, false},
	}
	for _, tc := range cases {
		result := truncateToWidth(tc.input, tc.width)
		w := lipgloss.Width(result)
		if tc.fits && result != tc.input {
			t.Errorf("expected %q unchanged at width %d, got %q", tc.input, tc.width, result)
		}
		if !tc.fits && w > tc.width {
			t.Errorf("truncateToWidth(%q, %d) width=%d exceeds max", tc.input, tc.width, w)
		}
	}
}
```

- [ ] **Step 2: Replace O(n²) truncateToWidth with ansi.Truncate**

```go
func truncateToWidth(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	return ansi.Truncate(s, maxW, "")
}
```

Add `"github.com/charmbracelet/x/ansi"` to imports in `list_pane.go`.

- [ ] **Step 3: Cache status bar rendering**

In `tui/statusbar.go`, add a cache to the status bar and only recompute when tickets change. Add `cachedBar string` and `cachedWidth int` fields or use a simple approach: compute in `SetTickets` and store.

- [ ] **Step 4: Run all TUI tests**

Run: `go test ./tui/ -v -count=1`

- [ ] **Step 5: Commit**

```bash
git add tui/list_pane.go tui/list_pane_test.go tui/statusbar.go tui/statusbar_test.go
git commit -m "perf: O(1) truncation via ansi.Truncate, cache status bar rendering"
```

---

### Task 9: Input Validation Hardening

**Files:**
- Modify: `server/server.go`
- Modify: `server/auth.go`
- Modify: `server/server_test.go`
- Modify: `server/auth_test.go`

- [ ] **Step 1: Write failing test — empty ticket title rejected**

```go
func TestServer_CreateTicket_RejectsEmptyTitle(t *testing.T) {
	// setup server, auth, workspace, board
	// POST ticket with empty title
	// expect 400
}
```

- [ ] **Step 2: Write failing test — username validated on member invite**

```go
func TestServer_InviteMember_ValidatesUsername(t *testing.T) {
	// POST member with username "bad user!!"
	// expect 400
}
```

- [ ] **Step 3: Write failing test — workspace/board name length limited**

```go
func TestServer_CreateWorkspace_RejectsLongName(t *testing.T) {
	// POST workspace with 200-char name
	// expect 400
}
```

- [ ] **Step 4: Implement validations**

In `server/server.go`:
- `createTicket`: Add `if input.Title == "" { return jsonErr(..., "title required") }`
- `addWorkspaceMember`: Add `if !validUsername.MatchString(input.Username) || len(input.Username) > 39 { return jsonErr(...) }`
- `createWorkspace` and `createBoard`: Add `if len(input.Name) > 100 { return jsonErr(...) }`

Extract status validation:
```go
func validateStatuses(statuses []string) error {
	for _, st := range statuses {
		if st == "" || strings.ContainsAny(st, ", ") {
			return fmt.Errorf("invalid status name: must be non-empty with no commas or spaces")
		}
	}
	return nil
}
```

Use in both `createBoard` and `updateBoard`.

- [ ] **Step 5: Fix swallowed error in auth.go IsWorkspaceMember**

```go
isMember, err := s.db.IsWorkspaceMember(input.Username)
if err != nil {
	return jsonErr(c, http.StatusInternalServerError, "internal server error")
}
```

- [ ] **Step 6: Run all tests**

Run: `go test ./... -count=1`

- [ ] **Step 7: Commit**

```bash
git add server/server.go server/auth.go server/server_test.go server/auth_test.go
git commit -m "fix: validate empty titles, member usernames, name lengths; handle DB errors"
```

---

### Task 10: HTTP Timeouts in CLI Commands

**Files:**
- Modify: `cmd/update.go`
- Modify: `cmd/login.go`

- [ ] **Step 1: Add httpClient helper with timeout**

In `cmd/update.go` (or a shared location):
```go
var httpClient = &http.Client{Timeout: 60 * time.Second}
```

- [ ] **Step 2: Replace all http.Get/http.Post with httpClient**

In `cmd/update.go`:
- Line 37: `httpClient.Get(dlURL)` instead of `http.Get(dlURL)`
- Line 82: `httpClient.Get(skillURL)` instead of `http.Get(skillURL)`

In `cmd/login.go`:
- Line 29: `httpClient.Post(serverURL+"/api/auth", ...)` instead of `http.Post(...)`

- [ ] **Step 3: Run build to verify**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git add cmd/update.go cmd/login.go
git commit -m "fix: add 60s timeout to HTTP clients in CLI commands"
```

---

### Task 11: Edit Command — Allow Clearing Fields

**Files:**
- Modify: `cmd/edit.go`

- [ ] **Step 1: Use Flags().Changed() instead of empty-string checks**

Replace:
```go
if editTitle != "" {
	fields["title"] = editTitle
}
```

With:
```go
if cmd.Flags().Changed("title") {
	fields["title"] = editTitle
}
if cmd.Flags().Changed("content") {
	fields["content"] = editContent
}
if cmd.Flags().Changed("assign") {
	fields["assignee"] = editAssign
}
```

- [ ] **Step 2: Run build to verify**

Run: `go build ./...`

- [ ] **Step 3: Commit**

```bash
git add cmd/edit.go
git commit -m "fix: allow clearing ticket fields via edit --assign=\"\""
```

---

### Task 12: DRY — Extract Shared Constants and Helpers

**Files:**
- Create: `model/shared.go`
- Modify: `cmd/update.go`
- Modify: `server/releases.go`
- Modify: `cmd/board.go`
- Modify: `cmd/list.go`
- Modify: `tui/styles.go`

- [ ] **Step 1: Create shared constants**

Create `model/shared.go`:
```go
package model

const GitHubRepo = "justaashir/raptor"
```

- [ ] **Step 2: Update cmd/update.go and server/releases.go to use shared constant**

Replace local `githubRepo` with `model.GitHubRepo`.

- [ ] **Step 3: Extract parseStatuses in cmd/board.go**

```go
func parseStatuses(raw string) []string {
	parts := strings.Split(raw, ",")
	var out []string
	for _, s := range parts {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
```

Use in both create and edit board commands.

- [ ] **Step 4: Run build and tests**

Run: `go build ./... && go test ./... -count=1`

- [ ] **Step 5: Commit**

```bash
git add model/shared.go cmd/update.go server/releases.go cmd/board.go
git commit -m "refactor: extract shared GitHubRepo constant and parseStatuses helper"
```

---

### Task 13: Remove Dead Code + Rename Cryptic Helpers

**Files:**
- Modify: `tui/styles.go:165-171`
- Modify: `tui/detail_pane.go:36-38`
- Modify: `tui/detail_pane_test.go`

- [ ] **Step 1: Remove DetailTitleStyle and DetailMetaKeyStyle from styles.go**

Delete the unused style declarations.

- [ ] **Step 2: Rename sp/bp/up in detail_pane.go**

```go
func strPtr(s string) *string  { return &s }
func boolPtr(b bool) *bool     { return &b }
func uintPtr(u uint) *uint     { return &u }
```

Update all call sites in the same file.

- [ ] **Step 3: Run tests**

Run: `go test ./tui/ -v -count=1`

- [ ] **Step 4: Commit**

```bash
git add tui/styles.go tui/detail_pane.go tui/detail_pane_test.go
git commit -m "cleanup: remove dead styles, rename sp/bp/up to strPtr/boolPtr/uintPtr"
```

---

### Task 14: Fix Error Handling — UserHomeDir

**Files:**
- Modify: `cmd/config.go:17-20`
- Modify: `tui/cache.go:16-19`

- [ ] **Step 1: Propagate UserHomeDir errors**

In `cmd/config.go`:
```go
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".raptor.json"), nil
}
```

Update `LoadConfig` and `SaveConfig` to handle the error from `configPath()`.

In `tui/cache.go`:
```go
func DefaultCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".raptor-cache.json"), nil
}
```

Update callers to handle the error.

- [ ] **Step 2: Update all callers**

Find and update all callers of `configPath()`, `LoadConfig()`, `SaveConfig()`, and `DefaultCachePath()`.

- [ ] **Step 3: Run all tests**

Run: `go test ./... -count=1`

- [ ] **Step 4: Commit**

```bash
git add cmd/config.go tui/cache.go cmd/root.go cmd/login.go tui/app.go
git commit -m "fix: propagate UserHomeDir errors instead of silently ignoring"
```

---

### Task 15: Server — Reduce DB Roundtrips in updateTicket

**Files:**
- Modify: `server/server.go:518-579`
- Modify: `server/db.go`

- [ ] **Step 1: Refactor updateTicket to reuse board from requireBoard**

The handler currently fetches the board in `requireBoard`, then fetches it AGAIN for status validation. Reuse the board returned by `requireBoard`:

```go
func (s *Server) updateTicket(c echo.Context) error {
	wid := c.Param("wid")
	bid := c.Param("bid")
	tid := c.Param("tid")
	if err := s.authorize(c, wid, "member"); err != nil {
		return jsonErr(c, http.StatusForbidden, err.Error())
	}
	board, err := s.requireBoard(c)
	if err != nil {
		return nil
	}

	// ... decode fields, validate status against board.StatusList() ...
	// Remove the second GetBoard call
}
```

- [ ] **Step 2: Remove double GetMemberRole in ListBoardsForUser**

In `server/db.go`, `ListBoardsForUser` calls `GetMemberRole` internally, but the handler already called `authorize` which also calls `GetMemberRole`. Refactor to accept a pre-authorized state or remove the internal check.

- [ ] **Step 3: Run all tests**

Run: `go test ./... -count=1`

- [ ] **Step 4: Commit**

```bash
git add server/server.go server/db.go
git commit -m "perf: reduce DB roundtrips in updateTicket and listBoards"
```

---

### Task 16: Test Coverage — Client Package

**Files:**
- Modify: `client/client_test.go`

- [ ] **Step 1: Add tests for untested client methods**

Add tests for: `DeleteWorkspace`, `ListWorkspaceMembers`, `InviteWorkspaceMember` (success + 409 conflict), `KickWorkspaceMember`, `GetBoard`, `ListBoards`, `DeleteBoard`, `GetTicket` (404 path), and `decode` (401 path).

Each test should use `httptest.NewServer` with a handler that returns the expected status/body.

- [ ] **Step 2: Run client tests**

Run: `go test ./client/ -v -cover`
Expected: Coverage should jump from 56.8% to ~85%+

- [ ] **Step 3: Commit**

```bash
git add client/client_test.go
git commit -m "test: add coverage for 9 untested client methods (401, 404, CRUD)"
```

---

### Task 17: Test Coverage — Server Handlers

**Files:**
- Modify: `server/server_test.go`
- Modify: `server/auth_test.go`
- Modify: `server/releases_test.go`

- [ ] **Step 1: Add WebSocket roundtrip integration test**

Test that a connected WS client receives broadcast when a ticket is created via the API.

- [ ] **Step 2: Add deleteBoard HTTP test**

- [ ] **Step 3: Add removeLastOwner rejection test**

- [ ] **Step 4: Add open registration test (empty allowedUsers, no members)**

- [ ] **Step 5: Add serverBaseURL branch tests**

Test: env var override, TLS detection, invalid host fallback.

- [ ] **Step 6: Run all server tests**

Run: `go test ./server/ -v -cover`
Expected: Coverage should increase from 75.2% to ~85%+

- [ ] **Step 7: Commit**

```bash
git add server/server_test.go server/auth_test.go server/releases_test.go
git commit -m "test: add WS roundtrip, deleteBoard, lastOwner, open registration, serverBaseURL tests"
```

---

### Task 18: Test Coverage — TUI and CMD

**Files:**
- Create: `cmd/root_test.go`
- Create: `cmd/config_test.go`
- Modify: `tui/app_test.go`
- Modify: `tui/styles_test.go`
- Modify: `tui/list_pane_test.go`

- [ ] **Step 1: Add cmd/config_test.go — config round-trip**

```go
func TestConfig_SaveLoad_RoundTrip(t *testing.T) {
	// Use temp dir, save config, load it back, compare
}
```

- [ ] **Step 2: Add cmd/root_test.go — requireWorkspace, requireBoard**

```go
func TestRequireWorkspace_ReturnsErrorWhenEmpty(t *testing.T) {
	// Test that empty workspace returns descriptive error
}
```

- [ ] **Step 3: Add TUI tests — board selector, StatusStar, narrow truncation**

- [ ] **Step 4: Run all tests with coverage**

Run: `go test ./... -cover`

- [ ] **Step 5: Commit**

```bash
git add cmd/root_test.go cmd/config_test.go tui/app_test.go tui/styles_test.go tui/list_pane_test.go
git commit -m "test: add cmd config/root tests, TUI board selector and style tests"
```

---

### Task 19: Final Verification + Push

**Files:** None

- [ ] **Step 1: Run full test suite**

```bash
go test ./... -v -count=1 -cover
```

- [ ] **Step 2: Build binary**

```bash
go build -o raptor .
```

- [ ] **Step 3: Run go vet**

```bash
go vet ./...
```

- [ ] **Step 4: Push branch**

```bash
git push -u origin fix/review-findings
```
