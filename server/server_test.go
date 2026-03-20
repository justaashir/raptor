package server

import (
	"encoding/json"
	"fmt"
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

// setupWorkspaceAndBoard creates a workspace and board via API, returns their IDs.
func setupWorkspaceAndBoard(t *testing.T, srv *Server, token string) (wsID, bdID string) {
	t.Helper()

	body := `{"name":"Team"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: create workspace failed: %d: %s", w.Code, w.Body.String())
	}
	var ws struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&ws)

	body = `{"name":"Board"}`
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/boards", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: create board failed: %d: %s", w.Code, w.Body.String())
	}
	var bd struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&bd)

	return ws.ID, bd.ID
}

func ticketURL(wsID, bdID string) string {
	return fmt.Sprintf("/api/workspaces/%s/boards/%s/tickets", wsID, bdID)
}

func TestServer_CreateAndListTickets(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := GenerateToken("alice", "secret")
	wsID, bdID := setupWorkspaceAndBoard(t, srv, token)

	body := `{"title":"Test ticket","content":"# Hello"}`
	req := httptest.NewRequest("POST", ticketURL(wsID, bdID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
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
	req = httptest.NewRequest("GET", ticketURL(wsID, bdID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
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
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := GenerateToken("alice", "secret")
	wsID, bdID := setupWorkspaceAndBoard(t, srv, token)

	ticket := model.NewTicket("Get me", "content", "alice")
	ticket.BoardID = bdID
	srv.db.CreateTicket(ticket)

	req := httptest.NewRequest("GET", ticketURL(wsID, bdID)+"/"+ticket.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
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
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := GenerateToken("alice", "secret")
	wsID, bdID := setupWorkspaceAndBoard(t, srv, token)

	ticket := model.NewTicket("Original", "", "alice")
	ticket.BoardID = bdID
	srv.db.CreateTicket(ticket)

	body := `{"title":"Updated","status":"done"}`
	req := httptest.NewRequest("PATCH", ticketURL(wsID, bdID)+"/"+ticket.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
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
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := GenerateToken("alice", "secret")
	wsID, bdID := setupWorkspaceAndBoard(t, srv, token)

	ticket := model.NewTicket("To delete", "", "alice")
	ticket.BoardID = bdID
	srv.db.CreateTicket(ticket)

	req := httptest.NewRequest("DELETE", ticketURL(wsID, bdID)+"/"+ticket.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
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
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := GenerateToken("alice", "secret")
	wsID, bdID := setupWorkspaceAndBoard(t, srv, token)

	req := httptest.NewRequest("GET", ticketURL(wsID, bdID)+"/nonexist", nil)
	req.Header.Set("Authorization", "Bearer "+token)
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

	body := `{"name":"Team A"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

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

func TestServer_Authorization_MemberCantCreateBoard(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice", "bob"})
	tokenAlice := GenerateToken("alice", "secret")
	tokenBob := GenerateToken("bob", "secret")

	body := `{"name":"Team"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var ws struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&ws)

	body = `{"username":"bob","role":"member"}`
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	body = `{"name":"Sprint"}`
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/boards", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenBob)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for member creating board, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_BoardScopedTickets(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := GenerateToken("alice", "secret")
	wsID, bdID := setupWorkspaceAndBoard(t, srv, token)

	body := `{"title":"Task 1"}`
	url := ticketURL(wsID, bdID)
	req := httptest.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var tickets []model.Ticket
	json.NewDecoder(w.Body).Decode(&tickets)
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(tickets))
	}
	if tickets[0].BoardID != bdID {
		t.Fatalf("expected board_id %q, got %q", bdID, tickets[0].BoardID)
	}
}

func TestServer_Auth_ChecksWorkspaceMembership(t *testing.T) {
	db, err := NewDB(":memory:", "alice", "bob")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	hub := NewHub()
	go hub.Run()
	srv := NewServer(db, hub, WithSecret("secret"))

	body := `{"username":"alice"}`
	req := httptest.NewRequest("POST", "/api/auth", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for alice, got %d: %s", w.Code, w.Body.String())
	}

	body = `{"username":"eve"}`
	req = httptest.NewRequest("POST", "/api/auth", strings.NewReader(body))
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for eve, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_WorkspaceMembers(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice", "bob"})
	tokenAlice := GenerateToken("alice", "secret")

	body := `{"name":"Team"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var ws struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&ws)

	body = `{"username":"bob","role":"admin"}`
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("GET", "/api/workspaces/"+ws.ID+"/members", nil)
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var members []struct{ Username string `json:"username"` }
	json.NewDecoder(w.Body).Decode(&members)
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}

	body = `{"role":"member"}`
	req = httptest.NewRequest("PATCH", "/api/workspaces/"+ws.ID+"/members/bob", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("DELETE", "/api/workspaces/"+ws.ID+"/members/bob", nil)
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestServer_WorkspaceInviteDuplicate(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice", "bob"})
	tokenAlice := GenerateToken("alice", "secret")

	body := `{"name":"Team"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var ws struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&ws)

	// First invite should succeed
	body = `{"username":"bob","role":"member"}`
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Second invite should return 409 Conflict
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate invite, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_BoardMembers(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice", "bob"})
	tokenAlice := GenerateToken("alice", "secret")
	tokenBob := GenerateToken("bob", "secret")

	body := `{"name":"Team"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var ws struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&ws)

	body = `{"username":"bob","role":"member"}`
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	body = `{"name":"Sprint"}`
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/boards", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var bd struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&bd)

	// bob (member) can't access board tickets yet
	url := ticketURL(ws.ID, bd.ID)
	req = httptest.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+tokenBob)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	// Grant bob board access
	body = `{"username":"bob"}`
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/boards/"+bd.ID+"/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Now bob can access
	req = httptest.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+tokenBob)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after grant, got %d: %s", w.Code, w.Body.String())
	}

	// Revoke bob's access
	req = httptest.NewRequest("DELETE", "/api/workspaces/"+ws.ID+"/boards/"+bd.ID+"/members/bob", nil)
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestServer_DeleteWorkspace_OwnerOnly(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice", "bob"})
	tokenAlice := GenerateToken("alice", "secret")
	tokenBob := GenerateToken("bob", "secret")

	body := `{"name":"Team"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var ws struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&ws)

	body = `{"username":"bob","role":"admin"}`
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// bob (admin) can't delete workspace
	req = httptest.NewRequest("DELETE", "/api/workspaces/"+ws.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenBob)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	// alice (owner) can delete
	req = httptest.NewRequest("DELETE", "/api/workspaces/"+ws.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_CreatedByFromAuth(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := GenerateToken("alice", "secret")
	wsID, bdID := setupWorkspaceAndBoard(t, srv, token)

	body := `{"title":"Auth ticket","content":""}`
	req := httptest.NewRequest("POST", ticketURL(wsID, bdID), strings.NewReader(body))
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

func TestServer_SearchTickets(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := GenerateToken("alice", "secret")
	wsID, bdID := setupWorkspaceAndBoard(t, srv, token)

	for _, title := range []string{"Fix login bug", "Add dashboard", "Update readme"} {
		body := fmt.Sprintf(`{"title":"%s"}`, title)
		req := httptest.NewRequest("POST", ticketURL(wsID, bdID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
	}

	req := httptest.NewRequest("GET", ticketURL(wsID, bdID)+"?q=login", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var tickets []model.Ticket
	json.NewDecoder(w.Body).Decode(&tickets)
	if len(tickets) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(tickets))
	}
	if tickets[0].Title != "Fix login bug" {
		t.Fatalf("expected 'Fix login bug', got %q", tickets[0].Title)
	}
}

func TestServer_ListTickets_ExcludesClosedByDefault(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := GenerateToken("alice", "secret")
	wsID, bdID := setupWorkspaceAndBoard(t, srv, token)

	// Create open ticket
	body := `{"title":"Open task"}`
	req := httptest.NewRequest("POST", ticketURL(wsID, bdID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Create and close a ticket
	body = `{"title":"Will close"}`
	req = httptest.NewRequest("POST", ticketURL(wsID, bdID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var closed model.Ticket
	json.NewDecoder(w.Body).Decode(&closed)

	body = `{"status":"closed","close_reason":"not needed"}`
	req = httptest.NewRequest("PATCH", ticketURL(wsID, bdID)+"/"+closed.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Default list should exclude closed
	req = httptest.NewRequest("GET", ticketURL(wsID, bdID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var tickets []model.Ticket
	json.NewDecoder(w.Body).Decode(&tickets)
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket (closed excluded), got %d", len(tickets))
	}

	// all=true should include closed
	req = httptest.NewRequest("GET", ticketURL(wsID, bdID)+"?all=true", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	json.NewDecoder(w.Body).Decode(&tickets)
	if len(tickets) != 2 {
		t.Fatalf("expected 2 tickets with all=true, got %d", len(tickets))
	}
}

func TestServer_ListMine(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice", "bob"})
	tokenAlice := GenerateToken("alice", "secret")
	tokenBob := GenerateToken("bob", "secret")

	// alice creates workspace+board
	wsID, bdID := setupWorkspaceAndBoard(t, srv, tokenAlice)

	// add bob to workspace and board
	body := `{"username":"bob","role":"admin"}`
	req := httptest.NewRequest("POST", "/api/workspaces/"+wsID+"/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	url := ticketURL(wsID, bdID)

	// Alice creates a ticket
	body = `{"title":"Alice ticket"}`
	req = httptest.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Bob creates a ticket
	body = `{"title":"Bob ticket"}`
	req = httptest.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenBob)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Alice lists mine
	req = httptest.NewRequest("GET", url+"?mine=true", nil)
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
