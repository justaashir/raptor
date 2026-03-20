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

func mustToken(t *testing.T, username, secret string) string {
	t.Helper()
	tok, err := GenerateToken(username, secret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	return tok
}

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
	token := mustToken(t, "alice", "secret")
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
	token := mustToken(t, "alice", "secret")
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
	token := mustToken(t, "alice", "secret")
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
	token := mustToken(t, "alice", "secret")
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
	token := mustToken(t, "alice", "secret")
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
	token := mustToken(t, "alice", "secret")

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
}

func TestServer_ListWorkspaces(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := mustToken(t, "alice", "secret")

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

	var workspaces []struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&workspaces)
	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(workspaces))
	}
}

func TestServer_Authorization_MemberCantCreateBoard(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice", "bob"})
	tokenAlice := mustToken(t, "alice", "secret")
	tokenBob := mustToken(t, "bob", "secret")

	body := `{"name":"Team"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var ws struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&ws)

	// Add bob as member
	body = `{"username":"bob"}`
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// bob (member) can't create board
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
	token := mustToken(t, "alice", "secret")
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
}

func TestServer_Auth_ChecksWorkspaceMembership(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})

	body := `{"username":"alice"}`
	req := httptest.NewRequest("POST", "/api/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for alice, got %d: %s", w.Code, w.Body.String())
	}

	body = `{"username":"eve"}`
	req = httptest.NewRequest("POST", "/api/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for eve, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_WorkspaceMembers(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice", "bob"})
	tokenAlice := mustToken(t, "alice", "secret")

	body := `{"name":"Team"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var ws struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&ws)

	body = `{"username":"bob"}`
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
	tokenAlice := mustToken(t, "alice", "secret")

	body := `{"name":"Team"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var ws struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&ws)

	body = `{"username":"bob"}`
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate invite, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_DeleteWorkspace_OwnerOnly(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice", "bob"})
	tokenAlice := mustToken(t, "alice", "secret")
	tokenBob := mustToken(t, "bob", "secret")

	body := `{"name":"Team"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var ws struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&ws)

	body = `{"username":"bob"}`
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// bob (member) can't delete workspace
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
	token := mustToken(t, "alice", "secret")
	wsID, bdID := setupWorkspaceAndBoard(t, srv, token)

	body := `{"title":"Auth ticket","content":""}`
	req := httptest.NewRequest("POST", ticketURL(wsID, bdID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var created model.Ticket
	json.NewDecoder(w.Body).Decode(&created)
	if created.CreatedBy != "alice" {
		t.Fatalf("expected created_by=alice, got %q", created.CreatedBy)
	}
}

func TestServer_SearchTickets(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := mustToken(t, "alice", "secret")
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
}

func TestServer_PatchRejectsInvalidStatus(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := mustToken(t, "alice", "secret")
	wsID, bdID := setupWorkspaceAndBoard(t, srv, token)

	body := `{"title":"Test"}`
	req := httptest.NewRequest("POST", ticketURL(wsID, bdID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var created model.Ticket
	json.NewDecoder(w.Body).Decode(&created)

	body = `{"status":"banana"}`
	req = httptest.NewRequest("PATCH", ticketURL(wsID, bdID)+"/"+created.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid status, got %d", w.Code)
	}
}

func TestServer_PatchStripsProtectedFields(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := mustToken(t, "alice", "secret")
	wsID, bdID := setupWorkspaceAndBoard(t, srv, token)

	body := `{"title":"Test"}`
	req := httptest.NewRequest("POST", ticketURL(wsID, bdID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var created model.Ticket
	json.NewDecoder(w.Body).Decode(&created)

	body = `{"id":"hacked","board_id":"other","created_by":"evil","title":"legit update"}`
	req = httptest.NewRequest("PATCH", ticketURL(wsID, bdID)+"/"+created.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var updated model.Ticket
	json.NewDecoder(w.Body).Decode(&updated)
	if updated.ID != created.ID {
		t.Fatalf("ID should not have changed, got %q", updated.ID)
	}
	if updated.Title != "legit update" {
		t.Fatalf("title should have been updated, got %q", updated.Title)
	}
}

func TestServer_TicketStats(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := mustToken(t, "alice", "secret")
	wsID, bdID := setupWorkspaceAndBoard(t, srv, token)

	for _, title := range []string{"Task 1", "Task 2"} {
		body := fmt.Sprintf(`{"title":"%s"}`, title)
		req := httptest.NewRequest("POST", ticketURL(wsID, bdID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
	}

	req := httptest.NewRequest("GET", ticketURL(wsID, bdID)+"?stats=true", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result map[string]any
	json.NewDecoder(w.Body).Decode(&result)
	total := result["total"].(float64)
	if total != 2 {
		t.Fatalf("expected total 2, got %v", total)
	}
	counts := result["counts"].(map[string]any)
	if counts["todo"].(float64) != 2 {
		t.Fatalf("expected 2 todo, got %v", counts["todo"])
	}
}

func TestServer_SkillEndpoint(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/skill", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Fatal("expected non-empty skill content")
	}
}

func TestServer_PatchOnlyProtectedFields_Returns400(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := mustToken(t, "alice", "secret")
	wsID, bdID := setupWorkspaceAndBoard(t, srv, token)

	body := `{"title":"Test"}`
	req := httptest.NewRequest("POST", ticketURL(wsID, bdID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var created model.Ticket
	json.NewDecoder(w.Body).Decode(&created)

	body = `{"id":"hacked","board_id":"other","created_by":"evil"}`
	req = httptest.NewRequest("PATCH", ticketURL(wsID, bdID)+"/"+created.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for all-protected-fields patch, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_ListMine(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice", "bob"})
	tokenAlice := mustToken(t, "alice", "secret")
	tokenBob := mustToken(t, "bob", "secret")

	wsID, bdID := setupWorkspaceAndBoard(t, srv, tokenAlice)

	// add bob to workspace
	body := `{"username":"bob"}`
	req := httptest.NewRequest("POST", "/api/workspaces/"+wsID+"/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	url := ticketURL(wsID, bdID)

	body = `{"title":"Alice ticket"}`
	req = httptest.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	body = `{"title":"Bob ticket"}`
	req = httptest.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenBob)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	req = httptest.NewRequest("GET", url+"?mine=true", nil)
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var tickets []model.Ticket
	json.NewDecoder(w.Body).Decode(&tickets)
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket for alice, got %d", len(tickets))
	}
}

func TestServer_CreateBoardWithCustomStatuses(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := mustToken(t, "alice", "secret")

	body := `{"name":"Team"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var ws struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&ws)

	body = `{"name":"Kanban","statuses":["backlog","active","review","shipped"]}`
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/boards", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var board model.Board
	json.NewDecoder(w.Body).Decode(&board)
	statuses := board.StatusList()
	if len(statuses) != 4 || statuses[0] != "backlog" || statuses[3] != "shipped" {
		t.Fatalf("expected custom statuses, got %v", statuses)
	}
}

func TestServer_UpdateBoard(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := mustToken(t, "alice", "secret")
	wsID, bdID := setupWorkspaceAndBoard(t, srv, token)

	body := `{"name":"Updated Board","statuses":["backlog","dev","done"]}`
	req := httptest.NewRequest("PATCH", "/api/workspaces/"+wsID+"/boards/"+bdID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var board model.Board
	json.NewDecoder(w.Body).Decode(&board)
	if board.Name != "Updated Board" {
		t.Fatalf("expected Updated Board, got %q", board.Name)
	}
	statuses := board.StatusList()
	if len(statuses) != 3 || statuses[0] != "backlog" {
		t.Fatalf("expected updated statuses, got %v", statuses)
	}
}

func TestServer_PatchValidatesStatusAgainstBoard(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := mustToken(t, "alice", "secret")

	// Create workspace
	body := `{"name":"Team"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var ws struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&ws)

	// Create board with custom statuses
	body = `{"name":"Board","statuses":["backlog","active","shipped"]}`
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/boards", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var bd struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&bd)

	// Create ticket
	body = `{"title":"Test"}`
	req = httptest.NewRequest("POST", ticketURL(ws.ID, bd.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var ticket model.Ticket
	json.NewDecoder(w.Body).Decode(&ticket)

	// "todo" is not a valid status on this board
	body = `{"status":"todo"}`
	req = httptest.NewRequest("PATCH", ticketURL(ws.ID, bd.ID)+"/"+ticket.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for status not in board, got %d", w.Code)
	}

	// "active" is valid
	body = `{"status":"active"}`
	req = httptest.NewRequest("PATCH", ticketURL(ws.ID, bd.ID)+"/"+ticket.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid board status, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_MemberSeesAllBoards(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice", "bob"})
	tokenAlice := mustToken(t, "alice", "secret")
	tokenBob := mustToken(t, "bob", "secret")

	body := `{"name":"Team"}`
	req := httptest.NewRequest("POST", "/api/workspaces/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var ws struct{ ID string `json:"id"` }
	json.NewDecoder(w.Body).Decode(&ws)

	// Add bob as member
	body = `{"username":"bob"}`
	req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Create two boards
	for _, name := range []string{"Board A", "Board B"} {
		body = fmt.Sprintf(`{"name":"%s"}`, name)
		req = httptest.NewRequest("POST", "/api/workspaces/"+ws.ID+"/boards", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenAlice)
		w = httptest.NewRecorder()
		srv.ServeHTTP(w, req)
	}

	// bob sees both boards (no board-level ACL)
	req = httptest.NewRequest("GET", "/api/workspaces/"+ws.ID+"/boards", nil)
	req.Header.Set("Authorization", "Bearer "+tokenBob)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var boards []model.Board
	json.NewDecoder(w.Body).Decode(&boards)
	if len(boards) != 2 {
		t.Fatalf("expected 2 boards for member, got %d", len(boards))
	}
}
